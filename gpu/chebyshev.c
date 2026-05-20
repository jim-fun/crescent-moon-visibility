#include "gpu/chebyshev.h"
#include <math.h>

#ifndef M_PI
#define M_PI 3.14159265358979323846
#endif

// Compute Chebyshev coefficients c[0..N] for f sampled at GL nodes x_k = cos(k*pi/N).
//
// Formula (standard for Chebyshev-Gauss-Lobatto interpolation):
//
//   c_j = (gamma_j / N) * sum_{k=0}^{N} (1/d_k) * f(x_k) * cos(j*k*pi/N)
//
//   gamma_j = 1 if j = 0 or j = N, else gamma_j = 2
//   d_k     = 2 if k = 0 or k = N, else d_k     = 1
//
// With these the resulting expansion f(x) = sum c_j T_j(x) interpolates
// f exactly at the N+1 GL nodes.
void cheb_compute_coeffs(int N, const double *values, double *coeffs) {
    for (int j = 0; j <= N; j++) {
        double sum = 0.0;
        for (int k = 0; k <= N; k++) {
            double w_k = (k == 0 || k == N) ? 0.5 : 1.0;
            sum += w_k * values[k] * cos((double)j * (double)k * M_PI / (double)N);
        }
        double gamma_over_N = (j == 0 || j == N) ? (1.0 / (double)N) : (2.0 / (double)N);
        coeffs[j] = gamma_over_N * sum;
    }
}

// Generate physical times of N+1 GL nodes mapped from [-1, +1] to [t_start, t_end].
void cheb_gl_nodes_in_range(int N, double t_start, double t_end, double *nodes) {
    double t_center   = 0.5 * (t_end + t_start);
    double t_halfspan = 0.5 * (t_end - t_start);
    for (int k = 0; k <= N; k++) {
        double x_k = cos((double)k * M_PI / (double)N);
        nodes[k] = t_center + t_halfspan * x_k;
    }
}
