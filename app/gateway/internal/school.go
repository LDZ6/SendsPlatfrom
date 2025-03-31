package internal

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/metadata"
	"platform/app/gateway/rpc"
	"platform/app/gateway/types"
)

// SchoolSchedule
// @Tags 学校
// @Summary 课表
// @Param token header string true "token"
// @Param scoreMessage body types.SemesterInfo true "获取学期。举例格式：2023-2024-1"
// @Success 200 {object} types.ResponseData
// @Failure 400 {object} types.ResponseData
// @Router /school/schedule [post]
func SchoolSchedule(c *gin.Context) {
	jsonValue := types.SemesterInfo{}
	c.ShouldBindJSON(&jsonValue)
	if _, exists := c.Get("stu_num"); !exists {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, errors.New("token无stu_num"))
		return
	}
	md := metadata.Pairs(
		"semester", jsonValue.Semester,
		"stu_num", c.Value("stu_num").(string),
	)
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	resp, err := rpc.SchoolScheduleRpc(ctx)
	if err != nil {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, err.Error())
		return
	}
	types.ResponseSuccess(c, resp)
}

// SchoolGpa
// @Tags 学校
// @Summary 绩点
// @Param token header string true "token"
// @Success 200 {object} types.ResponseData
// @Failure 400 {object} types.ResponseData
// @Router /school/gpa [get]
func SchoolGpa(c *gin.Context) {
	if _, exists := c.Get("stu_num"); !exists {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, errors.New("token无stu_num"))
		return
	}
	md := metadata.Pairs("stu_num", c.Value("stu_num").(string))
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	resp, err := rpc.SchoolGpaRpc(ctx)
	if err != nil {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, err.Error())
		return
	}
	types.ResponseSuccess(c, resp)
}

// SchoolGrade
// @Tags 学校
// @Summary 成绩
// @Param token header string true "token"
// @Success 200 {object} types.ResponseData
// @Failure 400 {object} types.ResponseData
// @Router /school/grade [get]
func SchoolGrade(c *gin.Context) {
	if _, exists := c.Get("stu_num"); !exists {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, errors.New("token无stu_num"))
		return
	}
	md := metadata.Pairs("stu_num", c.Value("stu_num").(string))
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	resp, err := rpc.SchoolGradeRpc(ctx)
	if err != nil {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, err.Error())
		return
	}
	types.ResponseSuccess(c, resp)
}

// SchoolXueFen
// @Tags 学校
// @Summary 学分
// @Param token header string true "token"
// @Success 200 {object} types.ResponseData
// @Failure 400 {object} types.ResponseData
// @Router /school/xuefen [get]
func SchoolXueFen(c *gin.Context) {
	if _, exists := c.Get("stu_num"); !exists {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, errors.New("token无stu_num"))
		return
	}
	md := metadata.Pairs("stu_num", c.Value("stu_num").(string))
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	resp, err := rpc.SchoolXuefenRpc(ctx)
	if err != nil {
		types.ResponseErrorWithMsg(c, types.CodeServerBusy, err.Error())
		return
	}
	types.ResponseSuccess(c, resp)
}
