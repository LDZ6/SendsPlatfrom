package tcc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// TXManager TCC事务管理器
type TXManager struct {
	ctx            context.Context
	stop           context.CancelFunc
	opts           *Options
	txStore        *TXStore
	registryCenter *registryCenter
	idem           *IdempotencyStore
}

// Options 配置选项
type Options struct {
	Timeout     time.Duration
	MonitorTick time.Duration
}

// Option 配置函数
type Option func(*Options)

// WithTimeout 设置超时时间
func WithTimeout(timeout time.Duration) Option {
	return func(opts *Options) {
		opts.Timeout = timeout
	}
}

// WithMonitorTick 设置监控间隔
func WithMonitorTick(tick time.Duration) Option {
	return func(opts *Options) {
		opts.MonitorTick = tick
	}
}

// NewTXManager 创建新的TXManager实例
func NewTXManager(txStore *TXStore, opts ...Option) *TXManager {
	ctx, cancel := context.WithCancel(context.Background())
	txManager := &TXManager{
		opts:           &Options{},
		txStore:        txStore,
		registryCenter: newRegistryCenter(),
		ctx:            ctx,
		stop:           cancel,
	}

	// use txStore's redis for idempotency tracking
	txManager.idem = NewIdempotencyStore(txStore.redis)

	for _, opt := range opts {
		opt(txManager.opts)
	}

	// 设置默认值
	repair(txManager.opts)

	// 启动监控协程
	go txManager.run()
	return txManager
}

// repair 修复配置选项
func repair(opts *Options) {
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.MonitorTick == 0 {
		opts.MonitorTick = 5 * time.Second
	}
}

// Stop 停止TXManager
func (t *TXManager) Stop() {
	t.stop()
}

// Register 注册TCC组件
func (t *TXManager) Register(component TCCComponent) error {
	return t.registryCenter.register(component)
}

// Transaction 执行TCC事务
func (t *TXManager) Transaction(ctx context.Context, reqs ...*RequestEntity) (string, bool, error) {
	tctx, cancel := context.WithTimeout(ctx, t.opts.Timeout)
	defer cancel()

	// 获取所有组件
	componentEntities, err := t.getComponents(tctx, reqs...)
	if err != nil {
		return "", false, err
	}

	// 创建事务记录
	txID, err := t.txStore.CreateTX(tctx, componentEntities.ToComponents()...)
	if err != nil {
		return "", false, err
	}

	// 执行两阶段提交
	success := t.twoPhaseCommit(ctx, txID, componentEntities)
	return txID, success, nil
}

// RequestEntity 请求实体
type RequestEntity struct {
	ComponentID string                 `json:"componentName"`
	Request     map[string]interface{} `json:"request"`
}

// ComponentEntities 组件实体列表
type ComponentEntities []*ComponentEntity

// ToComponents 转换为TCC组件列表
func (c ComponentEntities) ToComponents() []TCCComponent {
	components := make([]TCCComponent, 0, len(c))
	for _, entity := range c {
		components = append(components, entity.Component)
	}
	return components
}

// ComponentEntity 组件实体
type ComponentEntity struct {
	Request   map[string]interface{}
	Component TCCComponent
}

// run 运行监控协程
func (t *TXManager) run() {
	var tick time.Duration
	var err error
	for {
		// 如果出现了失败，tick 需要避让，遵循退避策略增大 tick 间隔时长
		if err == nil {
			tick = t.opts.MonitorTick
		} else {
			tick = t.backOffTick(tick)
		}
		select {
		case <-t.ctx.Done():
			return

		case <-time.After(tick):
			// 加锁，避免多个分布式多个节点的监控任务重复执行
			if err = t.txStore.Lock(t.ctx, t.opts.MonitorTick); err != nil {
				// 取锁失败时（大概率被其他节点占有），不对 tick 进行退避升级
				err = nil
				continue
			}

			// 获取仍然处于 hanging 状态的事务
			var txs []*Transaction
			if txs, err = t.txStore.GetHangingTXs(t.ctx); err != nil {
				_ = t.txStore.Unlock(t.ctx)
				continue
			}

			err = t.batchAdvanceProgress(txs)
			_ = t.txStore.Unlock(t.ctx)
		}
	}
}

// backOffTick 退避策略
func (t *TXManager) backOffTick(tick time.Duration) time.Duration {
	tick <<= 1
	if threshold := t.opts.MonitorTick << 3; tick > threshold {
		return threshold
	}
	return tick
}

// batchAdvanceProgress 批量推进事务进度
func (t *TXManager) batchAdvanceProgress(txs []*Transaction) error {
	// 对每笔事务进行状态推进
	errCh := make(chan error)
	go func() {
		// 并发执行，推进各比事务的进度
		var wg sync.WaitGroup
		for _, tx := range txs {
			// shadow
			tx := tx
			wg.Add(1)
			go func() {
				defer wg.Done()
				// 每个 goroutine 负责处理一笔事务
				if err := t.advanceProgress(tx); err != nil {
					// 遇到错误则投递到 errCh
					errCh <- err
				}
			}()
		}
		wg.Wait()
		close(errCh)
	}()

	var firstErr error
	// 通过 chan 阻塞在这里，直到所有 goroutine 执行完成，chan 被 close 才能往下
	for err := range errCh {
		// 记录遇到的第一个错误
		if firstErr != nil {
			continue
		}
		firstErr = err
	}

	return firstErr
}

// advanceProgressByTXID 根据事务ID推进进度
func (t *TXManager) advanceProgressByTXID(txID string) error {
	// 获取事务日志记录
	tx, err := t.txStore.GetTX(t.ctx, txID)
	if err != nil {
		return err
	}
	return t.advanceProgress(tx)
}

// advanceProgress 推进事务进度
func (t *TXManager) advanceProgress(tx *Transaction) error {
	// 根据各个 component try 请求的情况，推断出事务当前的状态
	txStatus := tx.getStatus(time.Now().Add(-t.opts.Timeout))
	// hanging 状态的暂时不处理
	if txStatus == "hanging" {
		return nil
	}

	// 根据事务是否成功，定制不同的处理函数
	success := txStatus == "successful"
	var confirmOrCancel func(ctx context.Context, component TCCComponent) (*TCCResp, error)
	var txAdvanceProgress func(ctx context.Context) error
	if success {
		confirmOrCancel = func(ctx context.Context, component TCCComponent) (*TCCResp, error) {
			// idempotent confirm with retry
			already, err := t.idem.CheckAndMark(ctx, tx.TXID, component.ID(), PhaseConfirm, 24*time.Hour)
			if err != nil {
				return nil, err
			}
			if already {
				return &TCCResp{ACK: true}, nil
			}
			return t.executeWithRetry(func(execCtx context.Context) (*TCCResp, error) {
				return component.Confirm(execCtx, tx.TXID)
			})
		}
		txAdvanceProgress = func(ctx context.Context) error {
			// 更新事务日志记录的状态为成功
			return t.txStore.TXSubmit(ctx, tx.TXID, true)
		}

	} else {
		confirmOrCancel = func(ctx context.Context, component TCCComponent) (*TCCResp, error) {
			// idempotent cancel with retry; support empty rollback (no temp state)
			already, err := t.idem.CheckAndMark(ctx, tx.TXID, component.ID(), PhaseCancel, 24*time.Hour)
			if err != nil {
				return nil, err
			}
			if already {
				return &TCCResp{ACK: true}, nil
			}
			// retry cancel; if component.Cancel returns error due to missing temp state, treat as success
			resp, err := t.executeWithRetry(func(execCtx context.Context) (*TCCResp, error) {
				return component.Cancel(execCtx, tx.TXID)
			})
			if err != nil {
				// best-effort empty rollback: mark success to avoid loops
				_ = t.idem.MarkExplicit(ctx, tx.TXID, component.ID(), PhaseCancel, 24*time.Hour)
				return &TCCResp{ACK: true}, nil
			}
			return resp, nil
		}

		txAdvanceProgress = func(ctx context.Context) error {
			// 更新事务日志记录的状态为失败
			return t.txStore.TXSubmit(ctx, tx.TXID, false)
		}
	}

	for _, component := range tx.Components {
		// 获取对应的 tcc component
		components, err := t.registryCenter.getComponents(component.ComponentID)
		if err != nil || len(components) == 0 {
			return errors.New("get tcc component failed")
		}
		// 执行二阶段的 confirm 或者 cancel 操作
		resp, err := confirmOrCancel(t.ctx, components[0])
		if err != nil {
			return err
		}
		if !resp.ACK {
			return fmt.Errorf("component: %s ack failed", component.ComponentID)
		}
	}

	// 二阶段操作都执行完成后，对事务状态进行提交
	return txAdvanceProgress(t.ctx)
}

// twoPhaseCommit 两阶段提交
func (t *TXManager) twoPhaseCommit(ctx context.Context, txID string, componentEntities ComponentEntities) bool {
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 并发执行，只要中间某次出现了失败，直接终止流程进行 cancel
	// 如果全量执行成功，则批量执行 confirm，然后返回成功的 ack，然后
	errCh := make(chan error, len(componentEntities))
	go func() {
		// 并发处理多个 component 的 try 流程
		var wg sync.WaitGroup
		for _, componentEntity := range componentEntities {
			// shadow
			componentEntity := componentEntity
			wg.Add(1)
			go func() {
				defer wg.Done()
				// strong idempotency for try
				already, ierr := t.idem.CheckAndMark(cctx, txID, componentEntity.Component.ID(), PhaseTry, 24*time.Hour)
				if ierr != nil {
					logrus.Errorf("tx try idempotency check failed, tx id: %s, component id: %s, err: %v", txID, componentEntity.Component.ID(), ierr)
					errCh <- ierr
					return
				}
				if already {
					// update tx as tried-success
					if uerr := t.txStore.TXUpdate(cctx, txID, componentEntity.Component.ID(), true); uerr != nil {
						logrus.Errorf("tx updated failed, tx id: %s, component id: %s, err: %v", txID, componentEntity.Component.ID(), uerr)
						errCh <- uerr
					}
					return
				}
				resp, err := t.executeWithRetry(func(execCtx context.Context) (*TCCResp, error) {
					return componentEntity.Component.Try(execCtx, &TCCReq{
						ComponentID: componentEntity.Component.ID(),
						TXID:        txID,
						Data:        componentEntity.Request,
					})
				})
				// 但凡有一个 component try 报错或者拒绝，都是需要进行 cancel 的，但会放在 advanceProgressByTXID 流程处理
				if err != nil || !resp.ACK {
					logrus.Errorf("tx try failed, tx id: %s, component id: %s, err: %v", txID, componentEntity.Component.ID(), err)
					// 对对应的事务进行更新
					if _err := t.txStore.TXUpdate(cctx, txID, componentEntity.Component.ID(), false); _err != nil {
						logrus.Errorf("tx updated failed, tx id: %s, component id: %s, err: %v", txID, componentEntity.Component.ID(), _err)
					}
					errCh <- fmt.Errorf("component: %s try failed", componentEntity.Component.ID())
					return
				}
				// try 请求成功，但是请求结果更新到事务日志失败时，也需要视为处理失败
				if err = t.txStore.TXUpdate(cctx, txID, componentEntity.Component.ID(), true); err != nil {
					logrus.Errorf("tx updated failed, tx id: %s, component id: %s, err: %v", txID, componentEntity.Component.ID(), err)
					errCh <- err
				}
			}()
		}

		wg.Wait()
		close(errCh)
	}()

	successful := true
	if err := <-errCh; err != nil {
		// 只要有一笔 try 请求出现问题，其他的都进行终止
		cancel()
		successful = false
	}

	// 执行二阶段. 即便第二阶段执行失败也无妨，可以通过轮询任务进行兜底处理
	if err := t.advanceProgressByTXID(txID); err != nil {
		logrus.Errorf("advance tx progress fail, txid: %s, err: %v", txID, err)
	}
	return successful
}

// executeWithRetry executes a TCC phase with retries and simple exponential backoff.
func (t *TXManager) executeWithRetry(fn func(ctx context.Context) (*TCCResp, error)) (*TCCResp, error) {
	var resp *TCCResp
	var err error
	backoff := 50 * time.Millisecond
	for attempt := 0; attempt < 3; attempt++ {
		resp, err = fn(t.ctx)
		if err == nil {
			return resp, nil
		}
		time.Sleep(backoff)
		// exponential backoff capped
		backoff <<= 1
		if backoff > 2*time.Second {
			backoff = 2 * time.Second
		}
	}
	return resp, err
}

// getComponents 获取组件
func (t *TXManager) getComponents(ctx context.Context, reqs ...*RequestEntity) (ComponentEntities, error) {
	if len(reqs) == 0 {
		return nil, errors.New("empty task")
	}

	// 调一下接口，确认这些都是合法的
	idToReq := make(map[string]*RequestEntity, len(reqs))
	componentIDs := make([]string, 0, len(reqs))
	for _, req := range reqs {
		if _, ok := idToReq[req.ComponentID]; ok {
			return nil, fmt.Errorf("repeat component: %s", req.ComponentID)
		}
		idToReq[req.ComponentID] = req
		componentIDs = append(componentIDs, req.ComponentID)
	}

	// 校验其合法性
	components, err := t.registryCenter.getComponents(componentIDs...)
	if err != nil {
		return nil, err
	}
	if len(componentIDs) != len(components) {
		return nil, errors.New("invalid componentIDs ")
	}

	entities := make(ComponentEntities, 0, len(components))
	for _, component := range components {
		entities = append(entities, &ComponentEntity{
			Request:   idToReq[component.ID()].Request,
			Component: component,
		})
	}

	return entities, nil
}

// getStatus 获取事务状态
func (tx *Transaction) getStatus(createdBefore time.Time) string {
	// 1 如果当中出现失败的，直接置为失败
	var hangingExist bool
	for _, component := range tx.Components {
		if component.TryStatus == "failure" {
			return "failure"
		}
		hangingExist = hangingExist || (component.TryStatus != "successful")
	}

	// 2 如果存在 hanging 状态，并且已经超时，也直接置为失败
	if hangingExist && tx.CreatedAt.Before(createdBefore) {
		return "failure"
	}

	// 3 如果存在组件 try 操作处于 hanging 状态，则返回 hanging 状态
	if hangingExist {
		return "hanging"
	}

	// 4 走到这个分支必然意味着所有组件的 try 操作都成功了
	return "successful"
}
