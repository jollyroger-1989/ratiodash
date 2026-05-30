package main

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"go.uber.org/fx/fxevent"
)

type fxLogrusLogger struct{}

func (l *fxLogrusLogger) LogEvent(event fxevent.Event) {
	switch e := event.(type) {
	case *fxevent.LoggerInitialized:
		if e.Err != nil {
			logrus.WithError(e.Err).WithField("constructor", e.ConstructorName).Error("fx_logger_init_failed")
			return
		}
		logrus.WithField("constructor", e.ConstructorName).Info("fx_logger_initialized")

	case *fxevent.Provided:
		entry := logrus.WithFields(logrus.Fields{
			"constructor": e.ConstructorName,
			"outputs":     strings.Join(e.OutputTypeNames, ","),
			"module":      e.ModuleName,
			"private":     e.Private,
		})
		if e.Err != nil {
			entry.WithError(e.Err).Error("fx_provide_failed")
			return
		}
		entry.Info("fx_provide")

	case *fxevent.Supplied:
		entry := logrus.WithFields(logrus.Fields{
			"type":   e.TypeName,
			"module": e.ModuleName,
		})
		if e.Err != nil {
			entry.WithError(e.Err).Error("fx_supply_failed")
			return
		}
		entry.Info("fx_supply")

	case *fxevent.Replaced:
		entry := logrus.WithFields(logrus.Fields{
			"outputs": strings.Join(e.OutputTypeNames, ","),
			"module":  e.ModuleName,
		})
		if e.Err != nil {
			entry.WithError(e.Err).Error("fx_replace_failed")
			return
		}
		entry.Info("fx_replace")

	case *fxevent.Decorated:
		entry := logrus.WithFields(logrus.Fields{
			"decorator": e.DecoratorName,
			"outputs":   strings.Join(e.OutputTypeNames, ","),
			"module":    e.ModuleName,
		})
		if e.Err != nil {
			entry.WithError(e.Err).Error("fx_decorate_failed")
			return
		}
		entry.Info("fx_decorate")

	case *fxevent.BeforeRun:
		logrus.WithFields(logrus.Fields{
			"kind":   e.Kind,
			"name":   e.Name,
			"module": e.ModuleName,
		}).Info("fx_before_run")

	case *fxevent.Invoking:
		logrus.WithFields(logrus.Fields{
			"function": e.FunctionName,
			"module":   e.ModuleName,
		}).Info("fx_invoking")

	case *fxevent.Invoked:
		entry := logrus.WithFields(logrus.Fields{
			"function": e.FunctionName,
			"module":   e.ModuleName,
		})
		if e.Err != nil {
			entry.WithError(e.Err).WithField("trace", e.Trace).Error("fx_invoke_failed")
			return
		}
		entry.Info("fx_invoked")

	case *fxevent.OnStartExecuting:
		logrus.WithFields(logrus.Fields{
			"function": e.FunctionName,
			"caller":   e.CallerName,
		}).Info("fx_on_start_executing")

	case *fxevent.OnStartExecuted:
		entry := logrus.WithFields(logrus.Fields{
			"function": e.FunctionName,
			"caller":   e.CallerName,
			"runtime":  e.Runtime.String(),
		})
		if e.Err != nil {
			entry.WithError(e.Err).Error("fx_on_start_failed")
			return
		}
		entry.Info("fx_on_start_executed")

	case *fxevent.OnStopExecuting:
		logrus.WithFields(logrus.Fields{
			"function": e.FunctionName,
			"caller":   e.CallerName,
		}).Info("fx_on_stop_executing")

	case *fxevent.OnStopExecuted:
		entry := logrus.WithFields(logrus.Fields{
			"function": e.FunctionName,
			"caller":   e.CallerName,
			"runtime":  e.Runtime.String(),
		})
		if e.Err != nil {
			entry.WithError(e.Err).Error("fx_on_stop_failed")
			return
		}
		entry.Info("fx_on_stop_executed")

	case *fxevent.Started:
		if e.Err != nil {
			logrus.WithError(e.Err).Error("fx_start_failed")
			return
		}
		logrus.Info("fx_started")

	case *fxevent.Stopped:
		if e.Err != nil {
			logrus.WithError(e.Err).Error("fx_stop_failed")
			return
		}
		logrus.Info("fx_stopped")

	case *fxevent.RollingBack:
		logrus.WithError(e.StartErr).Warn("fx_rolling_back")

	case *fxevent.RolledBack:
		if e.Err != nil {
			logrus.WithError(e.Err).Error("fx_rollback_failed")
			return
		}
		logrus.Warn("fx_rolled_back")

	case *fxevent.Stopping:
		logrus.WithField("signal", e.Signal.String()).Info("fx_stopping")

	case *fxevent.Run:
		entry := logrus.WithFields(logrus.Fields{
			"kind":    e.Kind,
			"name":    e.Name,
			"module":  e.ModuleName,
			"runtime": e.Runtime.String(),
		})
		if e.Err != nil {
			entry.WithError(e.Err).Error("fx_run_failed")
			return
		}
		entry.Info("fx_run")

	default:
		logrus.WithField("event_type", fmt.Sprintf("%T", event)).Info("fx_event")
	}
}
