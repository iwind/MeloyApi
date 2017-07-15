package MeloyApi

import (
	"net/http"
)

type HookManager struct {
	hooks []Hook
}

type Hook struct {
	BeforeFunc func (context *HookContext, next func())
	AfterFunc func (context *HookContext)

	IsAvailable bool
}

type HookContext struct {
	Writer http.ResponseWriter
	Request *http.Request
	Api *Api
	Response *http.Response
	Error error
}

// 转发请求之前调用
func (manager *HookManager) beforeHook(writer http.ResponseWriter, request *http.Request, api *Api, do func (context *HookContext)) {
	canDo := true

	context := &HookContext{}

	if len(manager.hooks) > 0 {
		context.Writer = writer
		context.Request = request
		context.Api = api
		for index, hook := range manager.hooks {
			if !canDo {
				hook.IsAvailable = false
				continue
			}

			canNext := false
			hook.BeforeFunc(context, func () {
				canNext = true
			})
			manager.hooks[index].IsAvailable = true
			if !canNext {
				canDo = false

				continue
			}
		}
	}

	if canDo {
		do(context)
	}
}

// 转发请求之后调用
func (manager *HookManager) afterHook(context *HookContext, response *http.Response, err error) {
	countHooks := len(manager.hooks)
	if countHooks > 0 {
		context.Response = response
		context.Error = err

		for i := countHooks - 1; i >= 0; i -- {
			hook := manager.hooks[i]
			if hook.IsAvailable {
				hook.AfterFunc(context)
			} else {
				break
			}
		}
	}

}

// 添加新钩子
func (manager *HookManager) AddHook(hook Hook)  {
	manager.hooks = append(manager.hooks, hook)
}