package pkg

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"math"
	"strconv"
	"time"
)

//对部分信息的简单校验（一旦触发直接封号，使其不管如何投掷都不记录信息）

// 定义一个结构体类型，用于存储骰子的参数
type Dice struct {
	Position   map[string]float64 `json:"position"`
	Quaternion []float64          `json:"quaternion"` // 旋转角度
}

// 计算四元数的模长
func quaternionNorm(q []float64) float64 {
	return math.Sqrt(math.Pow(q[0], 2) + math.Pow(q[1], 2) + math.Pow(q[2], 2) + math.Pow(q[3], 2))
}

// 校验是否有重复值
func hasDuplicateValue(m map[string]float64) bool {
	visited := make(map[float64]bool)

	for _, v := range m {
		if visited[v] {
			return true
		}
		visited[v] = true
	}

	return false
}

// 对参数进行简单校验
func ValidateParams(openid string, data []byte) (bool, error, []string) {
	today := time.Now().Day()
	var dices []Dice
	strs := make([]string, 0)
	err := json.Unmarshal(data, &dices)
	if err != nil {
		logrus.Info("[ValidERROR]:%v\n", err.Error())
		return false, err, nil
	}
	if len(dices) != 6 {
		logrus.Info("[ValidERROR]:%v\n", "骰子数量不对")
		return false, err, nil
	}
	for i, dice := range dices {
		// 检查位置是否有三个键值对
		if len(dice.Position) != 3 {
			return false, fmt.Errorf("invalid position length for dice %d", i+1), nil
		}
		if hasDuplicateValue(dice.Position) {
			return false, fmt.Errorf("hasDuplicateValue"), nil
		}
		// 检查旋转角度是否有四个元素
		if len(dice.Quaternion) != 4 {
			return false, fmt.Errorf("invalid quaternion length for dice %d", i+1), nil
		}
		bytes, _ := json.Marshal(dice.Quaternion)
		strs = append(strs, string(bytes)+openid+strconv.Itoa(today))
		// 检查旋转角度是否是单位四元数，即模长为1
		norm := quaternionNorm(dice.Quaternion)
		if math.Abs(norm-1) > 0.001 {
			return false, fmt.Errorf("invalid quaternion norm %f for dice %d", norm, i+1), nil
		}
	}

	// 如果没有出现错误，返回true
	return true, nil, strs
}
