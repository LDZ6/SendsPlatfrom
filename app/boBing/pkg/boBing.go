package pkg

type BoBingMessage struct {
	flag int
	// 1 跳猴
	// 2 虾米拢某
	// 3 一秀
	// 4 二举
	// 5.四进
	// 6.四进带二举
	// 7.四进带一秀
	// 8.三红
	// 9.对堂
	// 10.状元
	// 11.五子登科
	// 12.五子带一秀
	// 13.五红
	// 14.六杯黑
	// 15.六杯红
	// 16.状元插金花
}

func NewBoBingMessageTransform() *BoBingMessage {
	return &BoBingMessage{}
}

// set方法
func (b *BoBingMessage) SetType(flag int) {
	b.flag = flag
}

// 返回分数,类型,奖励次数
func (b *BoBingMessage) Transform() (score int, types string, count int) {
	switch b.flag {
	case 1:
		score = 0
		types = "跳猴"
	case 2:
		score = 0
		types = "虾米拢某"
	case 3:
		score = 1
		types = "一秀"
	case 4:
		score = 2
		types = "二举"
	case 5:
		score = 6
		types = "四进"
	case 6:
		score = 6
		types = "四进带二举"
	case 7:
		score = 6
		types = "四进带一秀"
	case 8:
		score = 8
		types = "三红"
	case 9:
		score = 10
		types = "对堂"
	case 10:
		score = 16
		types = "状元"
		count = 1
	case 11:
		score = 16
		types = "五子登科"
		count = 1
	case 12:
		score = 16
		types = "五子带一秀"
		count = 1
	case 13:
		score = 16
		types = "五红"
		count = 1
	case 14:
		score = 16
		types = "六杯黑"
		count = 2
	case 15:
		score = 16
		types = "六杯红"
		count = 2
	case 16:
		score = 16
		types = "状元插金花"
		count = 2
	}
	return
}
