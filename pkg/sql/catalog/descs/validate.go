// Copyright 2021 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package descs

import (
	"context"

	"github.com/cockroachdb/cockroach/pkg/clusterversion"
	"github.com/cockroachdb/cockroach/pkg/kv"
	"github.com/cockroachdb/cockroach/pkg/sql/catalog"
	"github.com/cockroachdb/cockroach/pkg/sql/catalog/descpb"
	"github.com/cockroachdb/cockroach/pkg/sql/catalog/internal/validate"
)

// Validate returns any descriptor validation errors after validating using the
// descriptor collection for retrieving referenced descriptors and namespace
// entries, if applicable.
func (tc *Collection) Validate(
	ctx context.Context,
	txn *kv.Txn,
	telemetry catalog.ValidationTelemetry,
	targetLevel catalog.ValidationLevel,
	descriptors ...catalog.Descriptor,
) (err error) {
	if !tc.validationModeProvider.ValidateDescriptorsOnRead() && !tc.validationModeProvider.ValidateDescriptorsOnWrite() {
		return nil
	}
	vd := tc.newValidationDereferencer(txn)
	version := tc.settings.Version.ActiveVersion(ctx)
	return validate.Validate(
		ctx,
		version,
		vd,
		telemetry,
		targetLevel,
		descriptors...).CombinedError()
}

// ValidateUncommittedDescriptors validates all uncommitted descriptors.
// Validation includes cross-reference checks. Referenced descriptors are
// read from the store unless they happen to also be part of the uncommitted
// descriptor set. We purposefully avoid using leased descriptors as those may
// be one version behind, in which case it's possible (and legitimate) that
// those are missing back-references which would cause validation to fail.
func (tc *Collection) ValidateUncommittedDescriptors(ctx context.Context, txn *kv.Txn) (err error) {
	if tc.skipValidationOnWrite || !tc.validationModeProvider.ValidateDescriptorsOnWrite() {
		return nil
	}
	var descs []catalog.Descriptor
	_ = tc.uncommitted.iterateUncommittedByID(func(desc catalog.Descriptor) error {
		descs = append(descs, desc)
		return nil
	})
	if len(descs) == 0 {
		return nil
	}
	return tc.Validate(ctx, txn, catalog.ValidationWriteTelemetry, validate.Write, descs...)
}

func (tc *Collection) newValidationDereferencer(txn *kv.Txn) validate.ValidationDereferencer {
	return &collectionBackedDereferencer{tc: tc, sd: tc.stored.NewValidationDereferencer(txn)}
}

// collectionBackedDereferencer wraps a Collection to implement the
// validate.ValidationDereferencer interface for validation.
type collectionBackedDereferencer struct {
	tc *Collection
	sd validate.ValidationDereferencer
}

var _ validate.ValidationDereferencer = &collectionBackedDereferencer{}

// DereferenceDescriptors implements the validate.ValidationDereferencer
// interface by leveraging the collection's uncommitted descriptors as well
// as its storage cache.
func (c collectionBackedDereferencer) DereferenceDescriptors(
	ctx context.Context, version clusterversion.ClusterVersion, reqs []descpb.ID,
) (ret []catalog.Descriptor, _ error) {
	ret = make([]catalog.Descriptor, len(reqs))
	fallbackReqs := make([]descpb.ID, 0, len(reqs))
	fallbackRetIndexes := make([]int, 0, len(reqs))
	for i, id := range reqs {
		if uc := c.tc.uncommitted.getUncommittedByID(id); uc == nil {
			fallbackReqs = append(fallbackReqs, id)
			fallbackRetIndexes = append(fallbackRetIndexes, i)
		} else {
			ret[i] = uc
		}
	}
	if len(fallbackReqs) == 0 {
		return ret, nil
	}
	fallbackRet, err := c.sd.DereferenceDescriptors(ctx, version, fallbackReqs)
	if err != nil {
		return nil, err
	}
	for j, desc := range fallbackRet {
		if desc != nil {
			ret[fallbackRetIndexes[j]] = desc
		}
	}
	return ret, nil
}

// DereferenceDescriptorIDs implements the validate.ValidationDereferencer
// interface by delegating to the storage cache.
func (c collectionBackedDereferencer) DereferenceDescriptorIDs(
	ctx context.Context, reqs []descpb.NameInfo,
) (ret []descpb.ID, _ error) {
	// TODO(postamar): namespace operations in general should go through Collection
	return c.sd.DereferenceDescriptorIDs(ctx, reqs)
}

// ValidateSelf validates that the descriptor is internally consistent.
// Validation may be skipped depending on mode.
func ValidateSelf(
	desc catalog.Descriptor,
	version clusterversion.ClusterVersion,
	dvmp DescriptorValidationModeProvider,
) error {
	if !dvmp.ValidateDescriptorsOnRead() && !dvmp.ValidateDescriptorsOnWrite() {
		return nil
	}
	return validate.Self(version, desc)
}
