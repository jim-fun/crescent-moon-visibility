# Crescent Moon Visibility Maps Generator — Build System
#
# Cross-platform compilation of CPU renderer, GPU renderer, and Go orchestrator.
#
# Platforms:
#   macOS  — clang/llvm, -framework OpenCL (Apple OpenCL / Metal backend)
#   Linux  — gcc/clang, -lOpenCL (AMD ROCm / NVIDIA CUDA / Intel GPU)
#
# Usage:
#   make              # Build everything (CPU + GPU + Go, GPU if headers available)
#   make gpu          # Build gpu_visibility.out
#   make cpu          # Build visibility.out (CPU)
#   make go           # Build crescent_maps (Go orchestrator)
#   make clean        # Remove all build artifacts

CC       ?= gcc
CFLAGS   := -O3 -Wall -Wextra -fno-exceptions
LDFLAGS  := -lm -I.
CPU_CFLAGS  := $(CFLAGS) -fopenmp -DPIXEL_PER_DEGREE_LON=4 -DPIXEL_PER_DEGREE_LAT=4
CPU_LDFLAGS  := $(LDFLAGS)
GPU_CFLAGS  := $(CFLAGS) -DPIXEL_PER_DEGREE_LON=10 -DPIXEL_PER_DEGREE_LAT=12

# Platform detection for OpenCL
UNAME_S := $(shell uname -s)
GPU_LDFLAGS := $(LDFLAGS)
ifeq ($(UNAME_S),Darwin)
  GPU_LDFLAGS += -framework OpenCL
else
  GPU_LDFLAGS += -lOpenCL
endif

GPU_BIN  := gpu_visibility.out
CPU_BIN  := visibility.out
GO_BIN   := crescent_maps

# Check for OpenCL dev headers
GPU_SUPPORTED := no
ifeq ($(UNAME_S),Darwin)
  GPU_SUPPORTED := yes
else
  ifneq ($(shell test -f /opt/rocm/include/CL/cl.h 2>/dev/null && echo yes || echo no), no)
    GPU_SUPPORTED := yes
    GPU_LDFLAGS += -L/opt/rocm/lib -lOpenCL
  else ifneq ($(shell test -d /usr/local/cuda && echo yes || echo no), no)
    GPU_SUPPORTED := yes
    GPU_LDFLAGS += -L/usr/local/cuda/lib64 -lOpenCL
  else ifneq ($(shell pkg-config --exists opencl 2>/dev/null && echo yes || echo no), no)
    GPU_SUPPORTED := yes
    GPU_LDFLAGS += $(shell pkg-config --cflags --libs opencl 2>/dev/null)
  else ifneq ($(shell test -f /usr/include/CL/cl.h 2>/dev/null && echo yes || echo no), no)
    GPU_SUPPORTED := yes
  endif
endif

.PHONY: all cpu gpu go clean

all: $(GO_BIN)
ifneq ($(GPU_SUPPORTED),yes)
	@echo "[warn] OpenCL dev headers not found — GPU renderer skipped (CPU + Go built OK)"
else
	@echo "[gpu] Building gpu_visibility.out ..."
	$(MAKE) gpu
endif

cpu: $(CPU_BIN)

$(CPU_BIN): cmd/visibility/visibility.cc thirdparty/astronomy.c
	$(CC) $(CPU_CFLAGS) -o $@ \
		-fopenmp \
		-DPIXEL_PER_DEGREE_LON=4 -DPIXEL_PER_DEGREE_LAT=4 \
		-I. \
		cmd/visibility/visibility.cc thirdparty/astronomy.c \
		$(CPU_LDFLAGS)

gpu: $(GPU_BIN)

$(GPU_BIN): gpu/gpu_render.c gpu/visibility_kernel.cl thirdparty/astronomy.c
	$(CC) $(GPU_CFLAGS) -o $@ \
		gpu/gpu_render.c thirdparty/astronomy.c \
		$(GPU_LDFLAGS)

go: $(GO_BIN)

$(GO_BIN): main.go internal/astro/astro.go go.mod
	go build -o $@ .

clean:
	rm -f $(CPU_BIN) $(GPU_BIN) $(GO_BIN)
