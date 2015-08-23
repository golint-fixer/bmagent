// Copyright (c) 2015 Monetas.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package powmgr_test

import (
	"io/ioutil"
	"sync"
	"testing"

	"github.com/monetas/bmclient/powmgr"
	"github.com/monetas/bmclient/store"
)

// Test the pow handler that runs the pow calculations.
func TestPowHandler(t *testing.T) {

	// It's pretty complicated to test running go routines.
	var powWait bool
	var mutex sync.RWMutex
	powChan := make(chan struct{})
	donePowChan := make(chan uint64)

	// A function that does not actually calculate the pow.
	mockPowFunc := func(target uint64, hash []byte) uint64 {
		mutex.RLock()
		pw := powWait
		mutex.RUnlock()
		if pw {
			<-powChan
			return 1
		}
		return 1
	}

	//A function that handles the completed pow.
	mockPowDone := func(index uint64, obj []byte) {
		donePowChan <- index
	}

	// Open store.
	f, err := ioutil.TempFile("", "tempstore")
	if err != nil {
		t.Fatal(err)
	}
	fName := f.Name()
	f.Close()

	pass := []byte("password")
	s, err := store.Open(fName, pass)
	if err != nil {
		t.Fatal(err)
	}

	testObj := [][]byte{
		[]byte("test0"),
		[]byte("test1"),
		[]byte("test2"),
		[]byte("test3"),
	}

	target := uint64(1152921504606846975)
	pm := powmgr.New(s.PowQueue, mockPowDone, mockPowFunc)

	// Test that an item can be added to the queue and will be run
	// once the queue handler is started.
	_, err = pm.RunPow(target, testObj[0])
	if err != nil {
		t.Error("Unable to submit to pow queue.")
	}
	pm.Start()
	test1 := <-donePowChan
	if test1 != 1 {
		t.Error("Incorrect test index returned.")
	}

	// Test that an item can be added to the queue after it is
	// running and that the item will be calculated.
	_, err = pm.RunPow(target, testObj[1])
	if err != nil {
		t.Error("Unable to submit to pow queue.")
	}
	test2 := <-donePowChan
	if test2 != 2 {
		t.Error("Incorrect test index returned.")
	}

	// Test that an item can be added to the queue while another
	// item is running and that it will be run eventually.
	mutex.Lock()
	powWait = true
	mutex.Unlock()

	pm.RunPow(target, testObj[2])
	pm.RunPow(target, testObj[3])

	mutex.Lock()
	powWait = false
	mutex.Unlock()

	powChan <- struct{}{}
	test3 := <-donePowChan
	if test3 != 3 {
		t.Error("Incorrect test index returned.")
	}
	test4 := <-donePowChan
	if test4 != 4 {
		t.Error("Incorrect test index returned.")
	}

	pm.Stop()
}