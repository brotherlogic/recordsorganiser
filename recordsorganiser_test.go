package main

import (
	"testing"
	"time"

	pb "github.com/brotherlogic/recordsorganiser/proto"
)

func TestMarkWithinQuota(t *testing.T) {
	s := getTestServer(".makrWithinQuota")
	c := &pb.Location{OverQuotaTime: time.Now().Unix()}
	s.markOverQuota(c, 0)

	if c.OverQuotaTime > 0 {
		t.Errorf("Quota has not been nulled out: %v", c)
	}
}
