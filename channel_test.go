/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package ttrpc

import (
	"bytes"
	"errors"
	"io"
	"net"
	"reflect"
	"testing"

	"google.golang.org/grpc/status"

	"google.golang.org/grpc/codes"
)

func TestReadWriteMessage(t *testing.T) {
	var (
		w, r     = net.Pipe()
		ch       = newChannel(w)
		rch      = newChannel(r)
		messages = [][]byte{
			[]byte("hello"),
			[]byte("this is a test"),
			[]byte("of message framing"),
		}
		received [][]byte
		errs     = make(chan error, 1)
	)

	go func() {
		for i, msg := range messages {
			if err := ch.send(uint32(i), 1, 0, msg); err != nil {
				errs <- err
				return
			}
		}

		w.Close()
	}()

	for {
		_, p, err := rch.recv()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				t.Fatal(err)
			}

			break
		}
		received = append(received, p)

		// make sure we don't have send errors
		select {
		case err := <-errs:
			if err != nil {
				t.Fatal(err)
			}
		default:
		}
	}

	if !reflect.DeepEqual(received, messages) {
		t.Fatalf("didn't received expected set of messages: %v != %v", received, messages)
	}

	select {
	case err := <-errs:
		if err != nil {
			t.Fatal(err)
		}
	default:
	}
}

func TestMessageOversize(t *testing.T) {
	var (
		w, _ = net.Pipe()
		wch  = newChannel(w)
		msg  = bytes.Repeat([]byte("a message of massive length"), 512<<10)
	)

	err := wch.send(1, 1, 0, msg)

	if status.Convert(err).Code() != codes.InvalidArgument {
		t.Fatalf("error expected while send a message of massive length")
	}
}
