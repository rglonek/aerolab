package aerospike

import "testing"

func TestParseNamePartsExporterArchAliases(t *testing.T) {
	cases := []struct {
		name string
		arch ArchitectureType
		os   OSName
		ft   FileType
	}{
		{"aerospike-prometheus-exporter_1.24.0_x86_64.tgz", ArchitectureTypeX86_64, OSNameUbuntu, FileTypeTGZ},
		{"aerospike-prometheus-exporter_1.24.0_aarch64.tgz", ArchitectureTypeAARCH64, OSNameUbuntu, FileTypeTGZ},
		{"aerospike-prometheus-exporter_1.24.0-1_amd64.deb", ArchitectureTypeX86_64, OSNameUbuntu, FileTypeDEB},
		{"aerospike-prometheus-exporter_1.24.0-1_arm64.deb", ArchitectureTypeAARCH64, OSNameUbuntu, FileTypeDEB},
		{"aerospike-prometheus-exporter-1.24.0-1.x86_64.rpm", ArchitectureTypeX86_64, OSNameCentOS, FileTypeRPM},
		{"aerospike-prometheus-exporter-1.24.0-1.aarch64.rpm", ArchitectureTypeAARCH64, OSNameCentOS, FileTypeRPM},
	}
	for _, tc := range cases {
		got := File{Name: tc.name}.ParseNameParts()
		if got == nil {
			t.Fatalf("%s: ParseNameParts returned nil", tc.name)
		}
		if got.Architecture != tc.arch {
			t.Fatalf("%s: arch=%s, want %s", tc.name, got.Architecture, tc.arch)
		}
		if got.OSName != tc.os {
			t.Fatalf("%s: os=%s, want %s", tc.name, got.OSName, tc.os)
		}
		if got.FileType != tc.ft {
			t.Fatalf("%s: type=%s, want %s", tc.name, got.FileType, tc.ft)
		}
		if got.ProductType != ProductTypeExporter {
			t.Fatalf("%s: product=%s, want exporter", tc.name, got.ProductType)
		}
	}
}

func TestParseNamePartsBackupArchAliases(t *testing.T) {
	cases := []struct {
		name string
		arch ArchitectureType
	}{
		{"aerospike-backup-service-3.1.0-1.x86_64.rpm", ArchitectureTypeX86_64},
		{"aerospike-backup-service-3.1.0-1.aarch64.rpm", ArchitectureTypeAARCH64},
		{"aerospike-backup-service_3.1.0_amd64.deb", ArchitectureTypeX86_64},
		{"aerospike-backup-service_3.1.0_arm64.deb", ArchitectureTypeAARCH64},
	}
	for _, tc := range cases {
		got := File{Name: tc.name}.ParseNameParts()
		if got == nil {
			t.Fatalf("%s: ParseNameParts returned nil", tc.name)
		}
		if got.Architecture != tc.arch {
			t.Fatalf("%s: arch=%s, want %s", tc.name, got.Architecture, tc.arch)
		}
	}
}
