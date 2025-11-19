package middlewares

import (
	"context"
	"fmt"
	"time"

	"platform/app/gateway/types"
	"platform/app/user/database/cache"
	"platform/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

func AuthUserCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		ctx := c.Request.Context()

		// 创建追踪span
		tracer := otel.Tracer("gateway")
		ctx, span := tracer.Start(ctx, "auth_user_check")
		defer span.End()

		// 创建日志上下文
		logger := utils.WithContext(ctx).WithRoute("auth_user_check")

		auth := c.GetHeader("token")
		if auth == "" {
			logger.Warn("用户认证失败：缺少token")
			span.SetAttributes(
				attribute.String("auth.error", "missing_token"),
				attribute.Bool("auth.success", false),
			)
			span.RecordError(fmt.Errorf("missing token"))
			c.Abort()
			types.ResponseErrorWithMsg(c, types.CodeUnauthorized, "Unauthorized")
			return
		}

		userClaims, err := utils.AnalyseUserToken(auth)
		if err != nil {
			logger.WithError(err).Warn("用户认证失败：token解析错误")
			span.SetAttributes(
				attribute.String("auth.error", "token_parse_error"),
				attribute.Bool("auth.success", false),
			)
			span.RecordError(err)
			c.Abort()
			types.ResponseErrorWithMsg(c, types.CodeUnauthorized, "Unauthorized")
			return
		}

		// 记录认证成功
		duration := time.Since(start)
		logger.WithUser(userClaims.OpenId).WithFields(logrus.Fields{
			"stu_num":     userClaims.StuNum,
			"duration_ms": duration.Milliseconds(),
		}).Info("用户认证成功")

		span.SetAttributes(
			attribute.String("auth.user_id", userClaims.OpenId),
			attribute.String("auth.stu_num", userClaims.StuNum),
			attribute.Bool("auth.success", true),
			attribute.Int64("auth.duration_ms", duration.Milliseconds()),
		)

		c.Set("open_id", userClaims.OpenId)
		c.Set("stu_num", userClaims.StuNum)
		c.Next()
	}
}

func AuthAdminCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		ctx := c.Request.Context()

		// 创建追踪span
		tracer := otel.Tracer("gateway")
		ctx, span := tracer.Start(ctx, "auth_admin_check")
		defer span.End()

		// 创建日志上下文
		logger := utils.WithContext(ctx).WithRoute("auth_admin_check")

		auth := c.GetHeader("token")
		if auth == "" {
			logger.Warn("管理员认证失败：缺少token")
			span.SetAttributes(
				attribute.String("auth.error", "missing_token"),
				attribute.Bool("auth.success", false),
			)
			span.RecordError(fmt.Errorf("missing token"))
			c.Abort()
			types.ResponseErrorWithMsg(c, types.CodeUnauthorized, "Unauthorized")
			return
		}

		adminClaims, err := utils.AnalyseAdminToken(auth)
		if err != nil {
			logger.WithError(err).Warn("管理员认证失败：token解析错误")
			span.SetAttributes(
				attribute.String("auth.error", "token_parse_error"),
				attribute.Bool("auth.success", false),
			)
			span.RecordError(err)
			c.Abort()
			types.ResponseErrorWithMsg(c, types.CodeUnauthorized, "Unauthorized")
			return
		}

		// 记录认证成功
		duration := time.Since(start)
		logger.WithUser(adminClaims.OpenId).WithFields(logrus.Fields{
			"organization": adminClaims.Organization,
			"duration_ms":  duration.Milliseconds(),
		}).Info("管理员认证成功")

		span.SetAttributes(
			attribute.String("auth.user_id", adminClaims.OpenId),
			attribute.Int("auth.organization", int(adminClaims.Organization)),
			attribute.Bool("auth.success", true),
			attribute.Int64("auth.duration_ms", duration.Milliseconds()),
		)

		c.Set("organization", adminClaims.Organization)
		c.Next()
	}
}

func AuthMassesCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		ctx := c.Request.Context()

		// 创建追踪span
		tracer := otel.Tracer("gateway")
		ctx, span := tracer.Start(ctx, "auth_masses_check")
		defer span.End()

		// 创建日志上下文
		logger := utils.WithContext(ctx).WithRoute("auth_masses_check")

		auth := c.GetHeader("token")
		if auth == "" {
			logger.Warn("群众认证失败：缺少token")
			span.SetAttributes(
				attribute.String("auth.error", "missing_token"),
				attribute.Bool("auth.success", false),
			)
			span.RecordError(fmt.Errorf("missing token"))
			c.Abort()
			types.ResponseErrorWithMsg(c, types.CodeUnauthorized, "Unauthorized")
			return
		}

		massesClaims, err := utils.AnalyseMassesToken(auth)
		if err != nil {
			logger.WithError(err).Warn("群众认证失败：token解析错误")
			span.SetAttributes(
				attribute.String("auth.error", "token_parse_error"),
				attribute.Bool("auth.success", false),
			)
			span.RecordError(err)
			c.Abort()
			types.ResponseErrorWithMsg(c, types.CodeUnauthorized, "Unauthorized")
			return
		}

		err = utils.ParseToken(auth)
		if err != nil {
			logger.WithUser(massesClaims.OpenId).WithError(err).Warn("群众认证失败：token过期")
			span.SetAttributes(
				attribute.String("auth.error", "token_expired"),
				attribute.Bool("auth.success", false),
			)
			span.RecordError(err)
			c.Abort()
			types.ResponseError(c, types.AuthenticationTimeout)

			// 清理缓存
			err = cache.NewRDBCache(c).Rdb.Unlink(massesClaims.OpenId).Err()
			if err != nil {
				logger.WithError(err).Error("清理用户缓存失败")
				types.ResponseErrorWithMsg(c, types.CodeServerBusy, err.Error())
				return
			}
			return
		}

		// 记录认证成功
		duration := time.Since(start)
		logger.WithUser(massesClaims.OpenId).WithFields(logrus.Fields{
			"nick_name":   massesClaims.NickName,
			"duration_ms": duration.Milliseconds(),
		}).Info("群众认证成功")

		span.SetAttributes(
			attribute.String("auth.user_id", massesClaims.OpenId),
			attribute.String("auth.nick_name", massesClaims.NickName),
			attribute.Bool("auth.success", true),
			attribute.Int64("auth.duration_ms", duration.Milliseconds()),
		)

		c.Set("open_id", massesClaims.OpenId)
		c.Set("nick_name", massesClaims.NickName)
		c.Next()
	}
}

// RequestTimeout wraps handlers with a context timeout and cancels downstream operations
func RequestTimeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
		if ctx.Err() == context.DeadlineExceeded {
			types.ResponseError(c, types.CodeTimeout)
			c.Abort()
			return
		}
	}
}
