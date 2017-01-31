// Copyright 2017 Monax Industries Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package loggers

import (
	"github.com/eris-ltd/eris-db/logging/structure"
	"github.com/eris-ltd/eris-db/logging/types"
	kitlog "github.com/go-kit/kit/log"
)

type infoTraceLogger struct {
	infoOnly                 *kitlog.Context
	infoAndTrace             *kitlog.Context
	infoOnlyOutputLogger     *kitlog.SwapLogger
	infoAndTraceOutputLogger *kitlog.SwapLogger
}

// Interface assertions
var _ types.InfoTraceLogger = (*infoTraceLogger)(nil)
var _ kitlog.Logger = (types.InfoTraceLogger)(nil)

// Create an InfoTraceLogger by passing the initial ouput loggers. The infoOnlyLogger will only be sent messages from
// the Info channel. The infoAndTraceLogger will be sent messages from both the Info and Trace channels.
func NewInfoTraceLogger(infoOnlyLogger, infoAndTraceLogger kitlog.Logger) types.InfoTraceLogger {
	// We will never halt the progress of a log emitter. If log output takes too
	// long will start dropping log lines by using a ring buffer.
	var infoOnlyOutputLogger, infoAndTraceOutputLogger kitlog.SwapLogger
	infoOnlyOutputLogger.Swap(infoOnlyLogger)
	infoAndTraceOutputLogger.Swap(infoAndTraceLogger)
	return &infoTraceLogger{
		infoOnlyOutputLogger:     &infoOnlyOutputLogger,
		infoAndTraceOutputLogger: &infoAndTraceOutputLogger,
		infoOnly: wrapOutputLogger(&infoOnlyOutputLogger).With(
			structure.ChannelKey, types.InfoChannelName,
			structure.LevelKey, types.InfoLevelName,
		),
		infoAndTrace: wrapOutputLogger(&infoAndTraceOutputLogger).With(
			structure.ChannelKey, types.TraceChannelName,
			structure.LevelKey, types.TraceLevelName,
		),
	}
}

func NewNoopInfoTraceLogger() types.InfoTraceLogger {
	return NewInfoTraceLogger(nil, nil)
}

func (l *infoTraceLogger) With(keyvals ...interface{}) types.InfoTraceLogger {
	return &infoTraceLogger{
		infoOnly:     l.infoOnly.With(keyvals...),
		infoAndTrace: l.infoAndTrace.With(keyvals...),
	}
}

func (l *infoTraceLogger) WithPrefix(keyvals ...interface{}) types.InfoTraceLogger {
	return &infoTraceLogger{
		infoOnly:     l.infoOnly.WithPrefix(keyvals...),
		infoAndTrace: l.infoAndTrace.WithPrefix(keyvals...),
	}
}

func (l *infoTraceLogger) Info(keyvals ...interface{}) error {
	// We log Info to the info only
	l.infoOnly.Log(keyvals...)
	// And pass to infoAndTrace
	l.infoAndTrace.Log(keyvals...)
	return nil
}

func (l *infoTraceLogger) Trace(keyvals ...interface{}) error {
	l.infoAndTrace.Log(keyvals...)
	return nil
}

func (l *infoTraceLogger) SwapInfoOnlyOutput(infoLogger kitlog.Logger) {
	l.infoOnlyOutputLogger.Swap(infoLogger)
}

func (l *infoTraceLogger) SwapInfoAndTraceOutput(infoTraceLogger kitlog.Logger) {
	l.infoAndTraceOutputLogger.Swap(infoTraceLogger)
}

// If logged to as a plain kitlog logger presume the message is for Trace
// This favours keeping Info reasonably quiet. Note that an InfoTraceLogger
// aware adapter can make its own choices, but we tend to think of logs from
// dependencies as less interesting than logs generated by us or specifically
// routed by us.
func (l *infoTraceLogger) Log(keyvals ...interface{}) error {
	l.Trace(keyvals...)
	return nil
}

// Wrap the output loggers with a a set of standard transforms, a non-blocking
// ChannelLogger and an outer context
func wrapOutputLogger(outputLogger kitlog.Logger) *kitlog.Context {
	return kitlog.NewContext(NonBlockingLogger(ErisFormatLogger(
		VectorValuedLogger(outputLogger))))
}
