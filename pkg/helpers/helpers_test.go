package helpers

import "testing"

func TestParseRequestURL(t *testing.T) {
	type args struct {
		requestURL string
	}
	tests := []struct {
		name            string
		args            args
		wantClusterID   string
		wantKubeAPIPath string
		wantErr         bool
	}{
		{
			name: "correct url input",
			args: args{
				requestURL: "127.0.0.1:8080/cluster1/api/pods?timeout=32s",
			},
			wantClusterID:   "cluster1",
			wantKubeAPIPath: "api/pods",
			wantErr:         false,
		},
		{
			name: "correct url input",
			args: args{
				requestURL: "127.0.0.1:8080/cluster1/api/pods",
			},
			wantClusterID:   "cluster1",
			wantKubeAPIPath: "api/pods",
			wantErr:         false,
		},
		{
			name: "wrong url input1",
			args: args{
				requestURL: "127.0.0.1:8080",
			},
			wantErr: true,
		},
		{
			name: "wrong url input",
			args: args{
				requestURL: "127.0.0.1:8080/cluster1",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotClusterID, gotKubeAPIPath, err := ParseRequestURL(tt.args.requestURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRequestURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotClusterID != tt.wantClusterID {
				t.Errorf("ParseRequestURL() gotClusterID = %v, want %v", gotClusterID, tt.wantClusterID)
			}
			if gotKubeAPIPath != tt.wantKubeAPIPath {
				t.Errorf("ParseRequestURL() gotKubeAPIPath = %v, want %v", gotKubeAPIPath, tt.wantKubeAPIPath)
			}
		})
	}
}
