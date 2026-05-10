package traceroute

import "testing"

func TestOptionsNormalize(t *testing.T) {
	opts := Options{}.Normalize()

	if opts.MaxHops != 64 {
		t.Fatalf("MaxHops = %d, want 64", opts.MaxHops)
	}
	if opts.QueriesPerHop != 3 {
		t.Fatalf("QueriesPerHop = %d, want 3", opts.QueriesPerHop)
	}
	if opts.PacketSize != 48 {
		t.Fatalf("PacketSize = %d, want 48", opts.PacketSize)
	}
}

func TestOptionsValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Options)
		wantErr bool
	}{
		{
			name:    "default",
			mutate:  func(o *Options) {},
			wantErr: false,
		},
		{
			name: "bad first hop",
			mutate: func(o *Options) {
				o.FirstHop = -1
			},
			wantErr: true,
		},
		{
			name: "max less than first",
			mutate: func(o *Options) {
				o.FirstHop = 10
				o.MaxHops = 5
			},
			wantErr: true,
		},
		{
			name: "bad packet size",
			mutate: func(o *Options) {
				o.PacketSize = -1
			},
			wantErr: true,
		},
		{
			name: "bad ip version",
			mutate: func(o *Options) {
				o.IPVersion = 99
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := DefaultOptions()
			tt.mutate(&opts)

			err := opts.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
