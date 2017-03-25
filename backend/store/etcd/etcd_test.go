package etcd

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"testing"

	"github.com/coreos/etcd/clientv3"
	"github.com/sensu/sensu-go/testing/util"
	"github.com/stretchr/testify/assert"
)

func TestNewEtcd(t *testing.T) {
	util.WithTempDir(func(tmpDir string) {
		ports := make([]int, 2)
		err := util.RandomPorts(ports)
		if err != nil {
			log.Panic(err)
		}
		clURL := fmt.Sprintf("http://127.0.0.1:%d", ports[0])
		apURL := fmt.Sprintf("http://127.0.0.1:%d", ports[1])
		initCluster := fmt.Sprintf("default=%s", apURL)
		fmt.Println(initCluster)

		cfg := NewConfig()
		cfg.StateDir = tmpDir
		cfg.ClientListenURL = clURL
		cfg.PeerListenURL = apURL
		cfg.InitialCluster = initCluster

		e, err := NewEtcd(cfg)
		if e != nil {
			defer e.Shutdown()
		}
		assert.NoError(t, err)
		if err != nil {
			pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
			pprof.Lookup("threadcreate").WriteTo(os.Stdout, 1)
			pprof.Lookup("heap").WriteTo(os.Stdout, 1)
			assert.FailNow(t, "unable to start new etcd")
		}

		client, err := e.NewClient()
		assert.NoError(t, err)
		kv := clientv3.NewKV(client)
		assert.NotNil(t, kv)

		putsResp, err := kv.Put(context.Background(), "key", "value")
		assert.NoError(t, err)
		assert.NotNil(t, putsResp)

		if putsResp == nil {
			assert.FailNow(t, "got nil put response from etcd")
		}

		getResp, err := kv.Get(context.Background(), "key")
		assert.NoError(t, err)
		assert.NotNil(t, getResp)

		if getResp == nil {
			assert.FailNow(t, "got nil get response from etcd")
		}
		assert.Equal(t, 1, len(getResp.Kvs))
		assert.Equal(t, "key", string(getResp.Kvs[0].Key))
		assert.Equal(t, "value", string(getResp.Kvs[0].Value))

		e.Shutdown()
	})
}