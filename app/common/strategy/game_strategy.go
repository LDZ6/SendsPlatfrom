package strategy

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"platform/utils"
)

// GameStrategy 游戏策略接口
type GameStrategy interface {
	// 计算分数
	CalculateScore(dice []Dice) (int, string, int)
	// 验证输入
	ValidateInput(data []byte) (bool, error)
	// 获取游戏类型
	GetGameType() string
	// 处理游戏逻辑
	ProcessGame(ctx context.Context, request interface{}) (interface{}, error)
}

// Dice 骰子结构
type Dice struct {
	Value int `json:"value"`
}

// BoBingStrategy 博饼策略实现
type BoBingStrategy struct {
	gameType string
}

// NewBoBingStrategy 创建博饼策略
func NewBoBingStrategy() *BoBingStrategy {
	return &BoBingStrategy{
		gameType: "bobing",
	}
}

// GetGameType 获取游戏类型
func (b *BoBingStrategy) GetGameType() string {
	return b.gameType
}

// CalculateScore 计算博饼分数
func (b *BoBingStrategy) CalculateScore(dice []Dice) (int, string, int) {
	if len(dice) != 6 {
		return 0, "invalid", 0
	}

	// 统计每个数字的出现次数
	counts := make(map[int]int)
	for _, d := range dice {
		counts[d.Value]++
	}

	// 博饼规则判断
	score, types, count := b.evaluateBoBing(counts)
	return score, types, count
}

// evaluateBoBing 评估博饼结果
func (b *BoBingStrategy) evaluateBoBing(counts map[int]int) (int, string, int) {
	// 统计各种组合
	fours := 0
	threes := 0
	pairs := 0
	singles := 0

	for _, count := range counts {
		switch count {
		case 6:
			return 100, "六红", 6
		case 5:
			return 50, "五红", 5
		case 4:
			fours++
		case 3:
			threes++
		case 2:
			pairs++
		case 1:
			singles++
		}
	}

	// 四红
	if fours == 1 {
		if threes == 1 {
			return 40, "四红带三红", 4
		}
		if pairs == 1 {
			return 35, "四红带对", 4
		}
		return 30, "四红", 4
	}

	// 三红
	if threes == 2 {
		return 25, "三红", 3
	}
	if threes == 1 {
		if pairs == 1 {
			return 20, "三红带对", 3
		}
		return 15, "三红", 3
	}

	// 对子
	if pairs == 3 {
		return 10, "三对", 2
	}
	if pairs == 2 {
		return 8, "两对", 2
	}
	if pairs == 1 {
		return 5, "一对", 2
	}

	// 顺子
	if b.isStraight(counts) {
		return 3, "顺子", 1
	}

	// 无奖
	return 0, "无奖", 0
}

// isStraight 检查是否为顺子
func (b *BoBingStrategy) isStraight(counts map[int]int) bool {
	// 检查1-6顺子
	straight1 := counts[1] == 1 && counts[2] == 1 && counts[3] == 1 &&
		counts[4] == 1 && counts[5] == 1 && counts[6] == 1

	// 检查2-6顺子
	straight2 := counts[2] == 1 && counts[3] == 1 && counts[4] == 1 &&
		counts[5] == 1 && counts[6] == 1 && counts[1] == 1

	return straight1 || straight2
}

// ValidateInput 验证输入
func (b *BoBingStrategy) ValidateInput(data []byte) (bool, error) {
	// 基本长度检查
	if len(data) < 10 {
		return false, errors.New("input too short")
	}

	// 检查是否包含有效字符
	validChars := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	for _, char := range data {
		if !contains(validChars, char) {
			return false, fmt.Errorf("invalid character: %c", char)
		}
	}

	return true, nil
}

// ProcessGame 处理游戏逻辑
func (b *BoBingStrategy) ProcessGame(ctx context.Context, request interface{}) (interface{}, error) {
	// 这里实现具体的博饼游戏逻辑
	// 包括验证、计分、记录等

	// 模拟处理时间
	time.Sleep(10 * time.Millisecond)

	return map[string]interface{}{
		"gameType":  b.gameType,
		"result":    "processed",
		"timestamp": time.Now().Unix(),
	}, nil
}

// contains 检查字符串是否包含字符
func contains(s string, char byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == char {
			return true
		}
	}
	return false
}

// GameStrategyFactory 游戏策略工厂
type GameStrategyFactory struct {
	strategies map[string]GameStrategy
	mutex      sync.RWMutex
}

// NewGameStrategyFactory 创建游戏策略工厂
func NewGameStrategyFactory() *GameStrategyFactory {
	factory := &GameStrategyFactory{
		strategies: make(map[string]GameStrategy),
	}

	// 注册默认策略
	factory.RegisterStrategy(NewBoBingStrategy())

	return factory
}

// RegisterStrategy 注册策略
func (f *GameStrategyFactory) RegisterStrategy(strategy GameStrategy) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.strategies[strategy.GetGameType()] = strategy
}

// GetStrategy 获取策略
func (f *GameStrategyFactory) GetStrategy(gameType string) (GameStrategy, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	if strategy, exists := f.strategies[gameType]; exists {
		return strategy, nil
	}

	return nil, fmt.Errorf("strategy for game type %s not found", gameType)
}

// ListStrategies 列出所有策略
func (f *GameStrategyFactory) ListStrategies() []string {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	var types []string
	for gameType := range f.strategies {
		types = append(types, gameType)
	}

	return types
}

// GameProcessor 游戏处理器
type GameProcessor struct {
	factory *GameStrategyFactory
}

// NewGameProcessor 创建游戏处理器
func NewGameProcessor() *GameProcessor {
	return &GameProcessor{
		factory: NewGameStrategyFactory(),
	}
}

// ProcessGame 处理游戏
func (gp *GameProcessor) ProcessGame(ctx context.Context, gameType string, request interface{}) (interface{}, error) {
	strategy, err := gp.factory.GetStrategy(gameType)
	if err != nil {
		return nil, utils.WrapError(err, utils.ErrCodeInvalidParam, "Invalid game type")
	}

	return strategy.ProcessGame(ctx, request)
}

// ValidateGameInput 验证游戏输入
func (gp *GameProcessor) ValidateGameInput(gameType string, data []byte) (bool, error) {
	strategy, err := gp.factory.GetStrategy(gameType)
	if err != nil {
		return false, err
	}

	return strategy.ValidateInput(data)
}

// CalculateGameScore 计算游戏分数
func (gp *GameProcessor) CalculateGameScore(gameType string, dice []Dice) (int, string, int, error) {
	strategy, err := gp.factory.GetStrategy(gameType)
	if err != nil {
		return 0, "", 0, err
	}

	score, types, count := strategy.CalculateScore(dice)
	return score, types, count, nil
}
