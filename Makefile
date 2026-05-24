# Crescent Moon Visibility Maps Generator — Build System
#
# Cross-platform compilation of CPU renderer, GPU renderer, and Go orchestrator.
#
# Platforms:
#   macOS  — clang/llvm, -framework OpenCL (Apple OpenCL / Metal backend).
#            Apple Silicon (M1+) automatically uses the FP32+DD kernel when
#            FP64 is unavailable (see gpu/visibility_kernel_fp32.cl).
#   Linux  — gcc/clang, -lOpenCL (AMD ROCm / NVIDIA CUDA / Intel GPU)
#
# Usage:
#   make              # Build everything into bin/ (CPU + GPU + Go)
#   make gpu          # Build bin/gpu_visibility.out
#   make cpu          # Build bin/visibility.out (CPU)
#   make go           # Build bin/crescent_maps (Go orchestrator)
#   make clean        # Remove all build artifacts from bin/
#
# All compiled outputs are placed in bin/ to keep the project root clean.

CC       ?= gcc
CFLAGS   := -O3 -Wall -Wextra -fno-exceptions
LDFLAGS  := -lm -I.
CPU_CFLAGS  := $(CFLAGS) -DPIXEL_PER_DEGREE_LON=10 -DPIXEL_PER_DEGREE_LAT=12
CPU_LDFLAGS  := $(LDFLAGS)
GPU_CFLAGS  := $(CFLAGS) -DPIXEL_PER_DEGREE_LON=10 -DPIXEL_PER_DEGREE_LAT=12

# Platform detection for OpenCL and OpenMP
UNAME_S := $(shell uname -s)
GPU_LDFLAGS := $(LDFLAGS)
ifeq ($(UNAME_S),Darwin)
  GPU_LDFLAGS += -framework OpenCL
  # Apple Clang doesn't accept -fopenmp directly. Use Homebrew's libomp
  # (brew install libomp) when available; otherwise build single-threaded.
  OMP_PREFIX := $(shell brew --prefix libomp 2>/dev/null)
  ifneq ($(OMP_PREFIX),)
    CPU_CFLAGS  += -Xpreprocessor -fopenmp -I$(OMP_PREFIX)/include
    CPU_LDFLAGS += -L$(OMP_PREFIX)/lib -lomp
  else
    $(warning [macOS] libomp not found via Homebrew. CPU renderer will be single-threaded. Install with: brew install libomp)
  endif
else
  GPU_LDFLAGS += -lOpenCL
  CPU_CFLAGS  += -fopenmp
endif

BIN_DIR  := bin
GPU_BIN  := $(BIN_DIR)/gpu_visibility.out
CPU_BIN  := $(BIN_DIR)/visibility.out
GO_BIN   := $(BIN_DIR)/crescent_maps

# Versioning (injected via ldflags)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildDate=$(DATE)"

# Check for OpenCL dev headers
GPU_SUPPORTED := no
ifeq ($(UNAME_S),Darwin)
  GPU_SUPPORTED := yes
else
  ifneq ($(shell test -f /opt/rocm/include/CL/cl.h 2>/dev/null && test -f /opt/rocm/lib/libOpenCL.so 2>/dev/null && echo yes || echo no), no)
    GPU_SUPPORTED := yes
    GPU_LDFLAGS += -L/opt/rocm/lib -lOpenCL
  else ifneq ($(shell test -f /usr/local/cuda/lib64/libOpenCL.so 2>/dev/null && echo yes || echo no), no)
    GPU_SUPPORTED := yes
    GPU_LDFLAGS += -L/usr/local/cuda/lib64 -lOpenCL
  else ifneq ($(shell pkg-config --exists opencl 2>/dev/null && echo yes || echo no), no)
    GPU_SUPPORTED := yes
    GPU_LDFLAGS += $(shell pkg-config --cflags --libs opencl 2>/dev/null)
  else ifneq ($(shell test -f /usr/include/CL/cl.h 2>/dev/null && echo yes || echo no), no)
    GPU_SUPPORTED := yes
    # Standard Debian/Ubuntu library locations
    ifeq ($(shell uname -m),aarch64)
      GPU_LDFLAGS += -L/lib/aarch64-linux-gnu -lOpenCL
    else
      GPU_LDFLAGS += -L/usr/lib/x86_64-linux-gnu -lOpenCL
    endif
  endif
endif

.PHONY: all cpu gpu go clean test

all: $(CPU_BIN) $(GO_BIN)
ifneq ($(GPU_SUPPORTED),yes)
	@echo "[warn] OpenCL dev headers not found — GPU renderer skipped (CPU + Go built OK)"
else
	@echo "[gpu] Building $(GPU_BIN) ..."
	$(MAKE) gpu
endif

cpu: $(CPU_BIN)

$(CPU_BIN): cmd/visibility/visibility.cc thirdparty/astronomy.c | $(BIN_DIR)
	g++ $(CPU_CFLAGS) -o $@ \
		-I. \
		cmd/visibility/visibility.cc thirdparty/astronomy.c \
		$(CPU_LDFLAGS)

gpu: $(GPU_BIN)

$(GPU_BIN): gpu/gpu_render.c gpu/chebyshev.c gpu/chebyshev.h gpu/visibility_kernel.cl thirdparty/astronomy.c | $(BIN_DIR)
	$(CC) $(GPU_CFLAGS) -o $@ \
		gpu/gpu_render.c gpu/chebyshev.c thirdparty/astronomy.c \
		$(GPU_LDFLAGS)

go: $(GO_BIN)

$(GO_BIN): main.go internal/astro/astro.go go.mod | $(BIN_DIR)
	go build $(LDFLAGS) -o $@ .

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

clean:
	rm -f $(CPU_BIN) $(GPU_BIN) $(GO_BIN)
	-rmdir $(BIN_DIR) 2>/dev/null || true

test:
	go test ./... -count=1

# Run the expensive renderer accuracy comparison (requires built renderers)
test-accuracy:
	RUN_ACCURACY_TEST=1 go test -v -run TestRendererAccuracy . -count=1
