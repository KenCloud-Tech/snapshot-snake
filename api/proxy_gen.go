// Code generated by github.com/londobell/tool/genapi. DO NOT EDIT.

package api

import (
	"context"
	"github.com/FIL_FIL_Snapshot/snapshot/saaf"
	"github.com/filecoin-project/lotus/chain/types"
	"golang.org/x/xerrors"
)

var ErrNotSupported = xerrors.New("method not supported")

type FilFilAPIStruct struct {
	Internal struct {
		ChainGetTipSet func(p0 context.Context, p1 types.TipSetKey) (*types.TipSet, error) ``

		FilFilDagExport func(p0 context.Context, p1 saaf.Height, p2 types.TipSetKey) (<-chan []byte, error) ``

		GetDagNode func(p0 context.Context, p1 saaf.Height) ([]saaf.Pointer, error) ``
	}
}

type FilFilAPIStub struct {
}

func (s *FilFilAPIStruct) ChainGetTipSet(p0 context.Context, p1 types.TipSetKey) (*types.TipSet, error) {
	if s.Internal.ChainGetTipSet == nil {
		return nil, ErrNotSupported
	}
	return s.Internal.ChainGetTipSet(p0, p1)
}

func (s *FilFilAPIStub) ChainGetTipSet(p0 context.Context, p1 types.TipSetKey) (*types.TipSet, error) {
	return nil, ErrNotSupported
}

func (s *FilFilAPIStruct) FilFilDagExport(p0 context.Context, p1 saaf.Height, p2 types.TipSetKey) (<-chan []byte, error) {
	if s.Internal.FilFilDagExport == nil {
		return nil, ErrNotSupported
	}
	return s.Internal.FilFilDagExport(p0, p1, p2)
}

func (s *FilFilAPIStub) FilFilDagExport(p0 context.Context, p1 saaf.Height, p2 types.TipSetKey) (<-chan []byte, error) {
	return nil, ErrNotSupported
}

func (s *FilFilAPIStruct) GetDagNode(p0 context.Context, p1 saaf.Height) ([]saaf.Pointer, error) {
	if s.Internal.GetDagNode == nil {
		return *new([]saaf.Pointer), ErrNotSupported
	}
	return s.Internal.GetDagNode(p0, p1)
}

func (s *FilFilAPIStub) GetDagNode(p0 context.Context, p1 saaf.Height) ([]saaf.Pointer, error) {
	return *new([]saaf.Pointer), ErrNotSupported
}

var _ FilFilAPI = new(FilFilAPIStruct)
