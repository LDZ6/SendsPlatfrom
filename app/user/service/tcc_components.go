package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"platform/app/common/tcc"
	"platform/app/user/database/cache"
	"platform/app/user/database/dao"
	"platform/app/user/database/models"
	"platform/utils"
	"platform/utils/school"
	"time"

	"github.com/redis/go-redis/v9"
)

// UserLoginTCCComponent 用户登录TCC组件
type UserLoginTCCComponent struct {
	userDao *dao.UserDao
	cache   *cache.RDBCache
	client  *redis.Client
}

func NewUserLoginTCCComponent(userDao *dao.UserDao, cache *cache.RDBCache, client *redis.Client) *UserLoginTCCComponent {
	return &UserLoginTCCComponent{
		userDao: userDao,
		cache:   cache,
		client:  client,
	}
}

func (u *UserLoginTCCComponent) ID() string {
	return "user_login_component"
}

func (u *UserLoginTCCComponent) Try(ctx context.Context, req *tcc.TCCReq) (*tcc.TCCResp, error) {
	// 解析请求数据
	code := req.Data["code"].(string)
	openId := req.Data["open_id"].(string)

	// 检查是否已经处理过
	key := "tcc_user_login:" + req.TXID + ":" + openId
	status, err := u.client.Get(ctx, key).Result()
	if err == nil && status == "tried" {
		return &tcc.TCCResp{ACK: true}, nil
	}

	// 检查用户是否已存在
	data, cnt, err := u.userDao.ExistUserByOpenid(openId)
	if err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	// 如果用户不存在，需要创建新用户
	if cnt == 0 {
		// 获取学号
		stResp, err := utils.GetStuNum(openId)
		if err != nil {
			return &tcc.TCCResp{ACK: false}, errors.New("找不到学号，请绑定桑梓微助手！")
		}

		// 获取微信用户信息
		wxLoginResp, err := utils.WXLogin(code)
		if err != nil {
			return &tcc.TCCResp{ACK: false}, err
		}

		wxInfoResp, err := utils.GetWeChatInfo(openId, wxLoginResp.AccessToken)
		if err != nil {
			return &tcc.TCCResp{ACK: false}, fmt.Errorf("%w %w", err, errors.New("获取头像昵称错误！"))
		}

		// 保存临时用户数据
		tempData := map[string]interface{}{
			"open_id":  openId,
			"stu_num":  stResp.Stuid,
			"avatar":   wxInfoResp.HeadImgURL,
			"nickname": wxInfoResp.Nickname,
			"is_admin": 0,
			"created":  true,
		}

		dataBytes, _ := json.Marshal(tempData)
		u.client.Set(ctx, key, string(dataBytes), 10*time.Minute)
		u.client.Set(ctx, key+":status", "tried", 10*time.Minute)
	} else {
		// 用户已存在，保存现有用户数据
		tempData := map[string]interface{}{
			"open_id":  openId,
			"stu_num":  data[0].StuNum,
			"avatar":   data[0].Avatar,
			"nickname": data[0].Nickname,
			"is_admin": data[0].IsAdmin,
			"created":  false,
		}

		dataBytes, _ := json.Marshal(tempData)
		u.client.Set(ctx, key, string(dataBytes), 10*time.Minute)
		u.client.Set(ctx, key+":status", "tried", 10*time.Minute)
	}

	return &tcc.TCCResp{ACK: true}, nil
}

func (u *UserLoginTCCComponent) Confirm(ctx context.Context, txID string) (*tcc.TCCResp, error) {
	// 获取临时数据
	key := "tcc_user_login:" + txID
	dataStr, err := u.client.Get(ctx, key).Result()
	if err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	var tempData map[string]interface{}
	if err := json.Unmarshal([]byte(dataStr), &tempData); err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	// 如果创建了新用户，执行数据库插入
	if tempData["created"].(bool) {
		user := models.User{
			OpenId:   tempData["open_id"].(string),
			StuNum:   tempData["stu_num"].(string),
			IsAdmin:  int(tempData["is_admin"].(float64)),
			Avatar:   tempData["avatar"].(string),
			Nickname: tempData["nickname"].(string),
		}

		err = u.userDao.CreateUser(&user)
		if err != nil {
			return &tcc.TCCResp{ACK: false}, err
		}
	}

	// 清理临时数据
	u.client.Del(ctx, key)
	u.client.Del(ctx, key+":status")

	return &tcc.TCCResp{ACK: true}, nil
}

func (u *UserLoginTCCComponent) Cancel(ctx context.Context, txID string) (*tcc.TCCResp, error) {
	// 获取临时数据
	key := "tcc_user_login:" + txID
	dataStr, err := u.client.Get(ctx, key).Result()
	if err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	var tempData map[string]interface{}
	if err := json.Unmarshal([]byte(dataStr), &tempData); err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	// 如果创建了新用户，需要删除
	if tempData["created"].(bool) {
		openId := tempData["open_id"].(string)
		// 这里需要实现删除用户的逻辑
		// u.userDao.DeleteUser(openId)
	}

	// 清理临时数据
	u.client.Del(ctx, key)
	u.client.Del(ctx, key+":status")

	return &tcc.TCCResp{ACK: true}, nil
}

// SchoolUserLoginTCCComponent 学校用户登录TCC组件
type SchoolUserLoginTCCComponent struct {
	userDao *dao.UserDao
	cache   *cache.RDBCache
	client  *redis.Client
}

func NewSchoolUserLoginTCCComponent(userDao *dao.UserDao, cache *cache.RDBCache, client *redis.Client) *SchoolUserLoginTCCComponent {
	return &SchoolUserLoginTCCComponent{
		userDao: userDao,
		cache:   cache,
		client:  client,
	}
}

func (s *SchoolUserLoginTCCComponent) ID() string {
	return "school_user_login_component"
}

func (s *SchoolUserLoginTCCComponent) Try(ctx context.Context, req *tcc.TCCReq) (*tcc.TCCResp, error) {
	// 解析请求数据
	code := req.Data["code"].(string)
	openId := req.Data["open_id"].(string)

	// 检查是否已经处理过
	key := "tcc_school_user_login:" + req.TXID + ":" + openId
	status, err := s.client.Get(ctx, key).Result()
	if err == nil && status == "tried" {
		return &tcc.TCCResp{ACK: true}, nil
	}

	// 获取学号
	stResp, err := utils.GetStuNum(openId)
	if err != nil {
		return &tcc.TCCResp{ACK: false}, errors.New("找不到学号，请绑定桑梓微助手！")
	}

	// 检查教务处凭证
	jwcCertificate := s.cache.GetJwcCertificate(stResp.Stuid)
	if jwcCertificate.Emaphome_WEU == "" {
		// 获取教务处凭证
		gsSession, err := school.GetGsSession(openId)
		if err != nil {
			return &tcc.TCCResp{ACK: false}, err
		}

		jwc := school.NewJwc()
		jwc.GsSession = gsSession
		emaphome_WEU, err := jwc.GetEmaphome_WEU()
		if err != nil {
			return &tcc.TCCResp{ACK: false}, err
		}

		// 保存临时凭证数据
		tempData := map[string]interface{}{
			"open_id":             openId,
			"stu_num":             stResp.Stuid,
			"gs_session":          gsSession,
			"emaphome_weu":        emaphome_WEU,
			"certificate_updated": true,
		}

		dataBytes, _ := json.Marshal(tempData)
		s.client.Set(ctx, key, string(dataBytes), 10*time.Minute)
		s.client.Set(ctx, key+":status", "tried", 10*time.Minute)
	} else {
		// 凭证已存在
		tempData := map[string]interface{}{
			"open_id":             openId,
			"stu_num":             stResp.Stuid,
			"gs_session":          jwcCertificate.GsSession,
			"emaphome_weu":        jwcCertificate.Emaphome_WEU,
			"certificate_updated": false,
		}

		dataBytes, _ := json.Marshal(tempData)
		s.client.Set(ctx, key, string(dataBytes), 10*time.Minute)
		s.client.Set(ctx, key+":status", "tried", 10*time.Minute)
	}

	return &tcc.TCCResp{ACK: true}, nil
}

func (s *SchoolUserLoginTCCComponent) Confirm(ctx context.Context, txID string) (*tcc.TCCResp, error) {
	// 获取临时数据
	key := "tcc_school_user_login:" + txID
	dataStr, err := s.client.Get(ctx, key).Result()
	if err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	var tempData map[string]interface{}
	if err := json.Unmarshal([]byte(dataStr), &tempData); err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	// 如果更新了凭证，保存到缓存
	if tempData["certificate_updated"].(bool) {
		jwcCertificate := cache.JwcCertificate{
			GsSession:    tempData["gs_session"].(string),
			Emaphome_WEU: tempData["emaphome_weu"].(string),
		}

		err = s.cache.SaveJwcCertificate(jwcCertificate, tempData["stu_num"].(string))
		if err != nil {
			return &tcc.TCCResp{ACK: false}, err
		}
	}

	// 清理临时数据
	s.client.Del(ctx, key)
	s.client.Del(ctx, key+":status")

	return &tcc.TCCResp{ACK: true}, nil
}

func (s *SchoolUserLoginTCCComponent) Cancel(ctx context.Context, txID string) (*tcc.TCCResp, error) {
	// 清理临时数据
	key := "tcc_school_user_login:" + txID
	s.client.Del(ctx, key)
	s.client.Del(ctx, key+":status")

	return &tcc.TCCResp{ACK: true}, nil
}

// MassesLoginTCCComponent 群众登录TCC组件
type MassesLoginTCCComponent struct {
	userDao *dao.UserDao
	cache   *cache.RDBCache
	client  *redis.Client
}

func NewMassesLoginTCCComponent(userDao *dao.UserDao, cache *cache.RDBCache, client *redis.Client) *MassesLoginTCCComponent {
	return &MassesLoginTCCComponent{
		userDao: userDao,
		cache:   cache,
		client:  client,
	}
}

func (m *MassesLoginTCCComponent) ID() string {
	return "masses_login_component"
}

func (m *MassesLoginTCCComponent) Try(ctx context.Context, req *tcc.TCCReq) (*tcc.TCCResp, error) {
	// 解析请求数据
	code := req.Data["code"].(string)
	openId := req.Data["open_id"].(string)

	// 检查是否已经处理过
	key := "tcc_masses_login:" + req.TXID + ":" + openId
	status, err := m.client.Get(ctx, key).Result()
	if err == nil && status == "tried" {
		return &tcc.TCCResp{ACK: true}, nil
	}

	// 检查用户是否已存在
	data, cnt, err := m.userDao.ExistUserByOpenid(openId)
	if err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	// 如果用户不存在，需要创建新用户
	if cnt == 0 {
		// 获取微信用户信息
		wxLoginResp, err := utils.WXLogin(code)
		if err != nil {
			return &tcc.TCCResp{ACK: false}, err
		}

		wxInfoResp, err := utils.GetWeChatInfo(openId, wxLoginResp.AccessToken)
		if err != nil {
			return &tcc.TCCResp{ACK: false}, fmt.Errorf("%w %w", err, errors.New("获取头像昵称错误！"))
		}

		// 保存临时用户数据
		tempData := map[string]interface{}{
			"open_id":  openId,
			"avatar":   wxInfoResp.HeadImgURL,
			"nickname": wxInfoResp.Nickname,
			"is_admin": 0,
			"created":  true,
		}

		dataBytes, _ := json.Marshal(tempData)
		m.client.Set(ctx, key, string(dataBytes), 10*time.Minute)
		m.client.Set(ctx, key+":status", "tried", 10*time.Minute)
	} else {
		// 用户已存在，保存现有用户数据
		tempData := map[string]interface{}{
			"open_id":  openId,
			"avatar":   data[0].Avatar,
			"nickname": data[0].Nickname,
			"is_admin": data[0].IsAdmin,
			"created":  false,
		}

		dataBytes, _ := json.Marshal(tempData)
		m.client.Set(ctx, key, string(dataBytes), 10*time.Minute)
		m.client.Set(ctx, key+":status", "tried", 10*time.Minute)
	}

	return &tcc.TCCResp{ACK: true}, nil
}

func (m *MassesLoginTCCComponent) Confirm(ctx context.Context, txID string) (*tcc.TCCResp, error) {
	// 获取临时数据
	key := "tcc_masses_login:" + txID
	dataStr, err := m.client.Get(ctx, key).Result()
	if err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	var tempData map[string]interface{}
	if err := json.Unmarshal([]byte(dataStr), &tempData); err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	// 如果创建了新用户，执行数据库插入
	if tempData["created"].(bool) {
		user := models.User{
			OpenId:   tempData["open_id"].(string),
			IsAdmin:  int(tempData["is_admin"].(float64)),
			Avatar:   tempData["avatar"].(string),
			Nickname: tempData["nickname"].(string),
		}

		err = m.userDao.CreateUser(&user)
		if err != nil {
			return &tcc.TCCResp{ACK: false}, err
		}
	}

	// 清理临时数据
	m.client.Del(ctx, key)
	m.client.Del(ctx, key+":status")

	return &tcc.TCCResp{ACK: true}, nil
}

func (m *MassesLoginTCCComponent) Cancel(ctx context.Context, txID string) (*tcc.TCCResp, error) {
	// 获取临时数据
	key := "tcc_masses_login:" + txID
	dataStr, err := m.client.Get(ctx, key).Result()
	if err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	var tempData map[string]interface{}
	if err := json.Unmarshal([]byte(dataStr), &tempData); err != nil {
		return &tcc.TCCResp{ACK: false}, err
	}

	// 如果创建了新用户，需要删除
	if tempData["created"].(bool) {
		openId := tempData["open_id"].(string)
		// 这里需要实现删除用户的逻辑
		// m.userDao.DeleteUser(openId)
	}

	// 清理临时数据
	m.client.Del(ctx, key)
	m.client.Del(ctx, key+":status")

	return &tcc.TCCResp{ACK: true}, nil
}
