# Second-Order Linear Homogeneous Equations

A second-order linear equation with constant coefficients and no forcing term is

$$a y'' + b y' + c y = 0$$

It is called **homogeneous** because the right-hand side is zero. (The word is also used for first-order equations of the form $y' = F(y/x)$, which are a different idea.)

## Characteristic equation

Try $y = e^{rx}$. Then $y' = r e^{rx}$ and $y'' = r^2 e^{rx}$, so the equation becomes $e^{rx}(a r^2 + b r + c) = 0$. Since $e^{rx}$ is never zero, $y = e^{rx}$ is a solution exactly when

$$a r^2 + b r + c = 0.$$

This is the **characteristic equation**. Because the equation is linear and homogeneous, any combination $C_1 y_1 + C_2 y_2$ of two independent solutions is again a solution, and every solution has that form.

**Example.** $y'' - 5y' + 6y = 0$ gives $r^2 - 5r + 6 = (r-2)(r-3) = 0$, so $r = 2$ or $r = 3$, and $y = C_1 e^{2x} + C_2 e^{3x}$.

## Three cases for the roots

The discriminant $b^2 - 4ac$ decides the shape of the general solution.

1. **Two distinct real roots** $r_1 \neq r_2$: $y = C_1 e^{r_1 x} + C_2 e^{r_2 x}$.
2. **One repeated root** $r$: $y = (C_1 + C_2 x) e^{r x}$. The factor $x$ supplies the missing second solution.
3. **Complex roots** $p \pm q i$: $y = e^{p x}(C_1 \cos q x + C_2 \sin q x)$. Euler's formula turns $e^{(p \pm qi)x}$ into real sines and cosines.

**Examples.** $y'' - 4y' + 4y = 0$: $(r - 2)^2 = 0$, so $y = (C_1 + C_2 x)e^{2x}$. $y'' + 2y' + 5y = 0$: $r = -1 \pm 2i$, so $y = e^{-x}(C_1 \cos 2x + C_2 \sin 2x)$.

## Initial value problems

Two conditions, $y(x_0)$ and $y'(x_0)$, fix $C_1$ and $C_2$. Write the general solution, differentiate it, substitute, and solve the two linear equations.

**Example.** $y'' - 5y' + 6y = 0$ with $y(0) = 1$, $y'(0) = 0$. From $y = C_1 e^{2x} + C_2 e^{3x}$: $C_1 + C_2 = 1$ and $2C_1 + 3C_2 = 0$, so $C_1 = 3$, $C_2 = -2$ and $y = 3e^{2x} - 2e^{3x}$.
