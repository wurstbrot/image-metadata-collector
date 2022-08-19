package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"sda.se/version-collector/internal/pkg/semantic"
)

func TestApplicationEntry_String(t *testing.T) {
	type fields struct {
		Name            string
		AppVersion      *semantic.Version
		HelmVersion     *semantic.Version
		IsManagedByHelm bool
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			"Full specification",
			fields{
				Name:            "bla",
				AppVersion:      &semantic.Version{Major: 1, Minor: 1, Patch: 1},
				HelmVersion:     &semantic.Version{Major: 2, Minor: 2, Patch: 2},
				IsManagedByHelm: true,
			},
			"bla: appVersion=1.1.1, helmVersion=2.2.2, managed-by-helm=true",
		},
		{
			"No Helm version",
			fields{
				Name:            "bla",
				AppVersion:      &semantic.Version{Major: 1, Minor: 1, Patch: 1},
				IsManagedByHelm: true,
			},
			"bla: appVersion=1.1.1, helmVersion=<nil>, managed-by-helm=true",
		},
		{
			"No app version",
			fields{
				Name:            "bla",
				HelmVersion:     &semantic.Version{Major: 2, Minor: 2, Patch: 2},
				IsManagedByHelm: true,
			},
			"bla: appVersion=<nil>, helmVersion=2.2.2, managed-by-helm=true",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := ApplicationEntry{
				Name:            tt.fields.Name,
				AppVersion:      tt.fields.AppVersion,
				HelmVersion:     tt.fields.HelmVersion,
				IsManagedByHelm: tt.fields.IsManagedByHelm,
			}
			assert.Equalf(t, tt.want, e.String(), "String()")
		})
	}
}
