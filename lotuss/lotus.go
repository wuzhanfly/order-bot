package lotuss

import (
	"context"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/client"
	"log"
	"net/http"
	"time"
)

var (
	lotusNode *node
)

type node struct {
	node   api.FullNode
	closer jsonrpc.ClientCloser
}

func Setup(url, authToken string) error {
	var err error
	lotusNode, err = getNode(url, authToken)
	if err != nil {
		return err
	}
	return err
}
func Node() api.FullNode {
	return lotusNode.node
}
func getNode(url, authToken string) (*node, error) {
	headers := http.Header{"Authorization": []string{"Bearer " + authToken}}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*6)
	defer cancel()

	addr := "ws://" + url + "/rpc/v1"
	lotus, closer, err := client.NewFullNodeRPCV1(ctx, addr, headers)
	if err != nil {
		log.Fatalf("[lotus] get lotus client from node[%s] err: %s", url, err.Error())
	}
	return &node{
		node:   lotus,
		closer: closer,
	}, nil

}
