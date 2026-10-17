# Optimg

Simple command-line utility for optimising image files.\
It will perform the following actions on each .jpeg, .jpg, and .png file in the
current directory and every subdirectory:
1. Convert .jpeg to .png
2. Resize enormous files (>6k resolution) using IrfanView's best-quality
resampling function
3. run OptiPNG (lossless compression)

Due to OptiPNG's limits, it is *not* concurrent.

## Dependencies

Dependencies must be installed on your PATH:\
[OptiPNG](https://optipng.sourceforge.net/)\
[IrfanView (64-bit)](https://www.irfanview.com/64bit.htm)

## Compile

```
go build
```

## Usage
```
$ optimg [OPTIONS]

Options:
-h                          Print help information and exit.

-o <optimisation level>     Matching OptiPNG's levels, this is passed directly on.
                            Must be a number in the range 0-7 inclusive. Higher numbers
                            optimise better (with diminishing returns) but the process
                            takes longer. Default value: 2.

-min <integer>              An image with a dimension below this number of pixels
                            will never be resized. Default value: 4000.

-max <integer>              An image with a dimension above this number of pixels
                            will be resized (unless its other dimension is below
                            the value of -min). Default value: 6000.
```