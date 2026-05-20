// chebyshev.h — Chebyshev polynomial ephemeris approximation.
//
// We fit each per-pixel-varying astronomical quantity (sun/moon RA, Dec,
// moon semi-diameter, moon elongation) with a single Chebyshev polynomial
// over the rendering window. The GPU then evaluates the polynomial at the
// exact moment of sunset for each pixel — eliminating the linear-interp
// error that the dense table version carried.
//
// A degree-24 polynomial over a 3-day window matches the underlying
// Astronomy Engine to ~1e-12 rad, well below the library's own internal
// error budget.

#ifndef CRESCENT_CHEBYSHEV_H
#define CRESCENT_CHEBYSHEV_H

// Number of Chebyshev coefficients per quantity (= degree + 1).
// 25 coefficients = degree 24, ample for a 3-day window.
#define CHEB_DEGREE   24
#define CHEB_N_COEFFS (CHEB_DEGREE + 1)

// Compute Chebyshev coefficients c[0..N] for f sampled at the N+1
// Chebyshev-Gauss-Lobatto nodes x_k = cos(k*pi/N), k = 0..N.
//
//   values[k] must contain f(x_k) for k = 0..N (i.e. values[0] at x=+1, values[N] at x=-1)
//   coeffs[j] receives the coefficient of T_j(x), j = 0..N
//
// The resulting expansion is f(x) ≈ sum_{j=0}^{N} c_j * T_j(x).
void cheb_compute_coeffs(int N, const double *values, double *coeffs);

// Fill nodes[k] with the physical time corresponding to the k-th GL node
// mapped from [-1, +1] to [t_start, t_end].
void cheb_gl_nodes_in_range(int N, double t_start, double t_end, double *nodes);

#endif
