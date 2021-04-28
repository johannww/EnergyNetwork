// Copyright the Hyperledger Fabric contributors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package shim

import (
	pb "github.com/hyperledger/fabric-protos-go/peer"
)

const (
	// OK constant - status code less than 400, endorser will endorse it.
	// OK means init or invoke successfully.
	OK = 200

	// ERRORTHRESHOLD constant - status code greater than or equal to 400 will be considered an error and rejected by endorser.
	ERRORTHRESHOLD = 400

	// ERROR constant - default error value
	ERROR = 500
)

// Success ...
func Success(payload []byte) pb.Response {
	return pb.Response{
		Status:  OK,
		Payload: payload,
		//JOHANN PRIORITY
		Priority:               pb.Priority_MEDIUM,
		BypassPhantomReadCheck: false,
	}
}

//SuccessWithPriority with setting transaction priority
func SuccessWithPriority(payload []byte, transactionPriority pb.Priority) pb.Response {
	return pb.Response{
		Status:  OK,
		Payload: payload,
		//JOHANN PRIORITY
		Priority:               transactionPriority,
		BypassPhantomReadCheck: false,
	}
}

//SuccessWithPriorityBypassPhantomReadCheck with setting transaction priority and
//bypassing PHANTOM_READ_CONFLICT verification at commit time
func SuccessWithPriorityBypassPhantomReadCheck(payload []byte, transactionPriority pb.Priority) pb.Response {
	response := SuccessWithPriority(payload, transactionPriority)
	response.BypassPhantomReadCheck = true
	return response
}

// Error ...
func Error(msg string) pb.Response {
	return pb.Response{
		Status:                 ERROR,
		Message:                msg,
		BypassPhantomReadCheck: false,
	}
}
