package main

import (
	"github.com/keybase/protocol/go"
	fmprpc "github.com/maxtaco/go-framed-msgpack-rpc"
	"github.com/ugorji/go/codec"
	"net"
	"net/rpc"
)

var __cli *rpc.Client
var __srv *rpc.Server

func GetRpcClient() (ret *rpc.Client, conn net.Conn, err error) {
	if __cli != nil {
	} else if conn, err = G.GetSocket(); err == nil {
		var mh codec.MsgpackHandle
		cdc := fmprpc.MsgpackSpecRpc.ClientCodec(conn, &mh, true)
		__cli = rpc.NewClientWithCodec(cdc)
	}
	ret = __cli
	return
}

func GetRpcServer() (ret *rpc.Server, conn net.Conn, err error) {
	if __srv != nil {
	} else if conn, err = G.GetSocket(); err == nil {
		__srv = rpc.NewServer()
		var mh codec.MsgpackHandle
		rpcCodec := fmprpc.MsgpackSpecRpc.ServerCodec(conn, &mh, true)
		__srv.ServeCodec(rpcCodec)
	}
	ret = __srv
	return
}

func GetSignupClient() (cli keybase_1.SignupClient, err error) {
	var rpc *rpc.Client
	if rpc, _, err = GetRpcClient(); err == nil {
		cli = keybase_1.SignupClient{rpc}
	}
	return
}

func GetConfigClient() (cli keybase_1.ConfigClient, err error) {
	var rpc *rpc.Client
	if rpc, _, err = GetRpcClient(); err == nil {
		cli = keybase_1.ConfigClient{rpc}
	}
	return
}

func GetLoginClient(gen func(con net.Conn) keybase_1.LoginUiInterface) (cli keybase_1.LoginClient, err error) {
	var rcli *rpc.Client
	var srv *rpc.Server
	var con net.Conn
	if rcli, _, err = GetRpcClient(); err != nil {
		return
	}
	cli = keybase_1.LoginClient{rcli}
	if gen == nil {
		return
	}
	if srv, con, err = GetRpcServer(); err == nil {
		keybase_1.RegisterLoginUi(srv, gen(con))
	}
	return
}

// TrackClients also act as TrackServers since they have to reply
// to UI events. So we allocate both at once.
func GetIdentifyClient(gen func(con net.Conn) keybase_1.IdentifyUiInterface) (cli keybase_1.IdentifyClient, err error) {
	var rcli *rpc.Client
	var srv *rpc.Server
	var con net.Conn
	if rcli, _, err = GetRpcClient(); err != nil {
		return
	}
	cli = keybase_1.IdentifyClient{rcli}
	if gen == nil {
		return
	}
	if srv, con, err = GetRpcServer(); err == nil {
		keybase_1.RegisterIdentifyUi(srv, gen(con))
	}
	return
}
