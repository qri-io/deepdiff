<a name="v0.2.0"></a>
# [v0.2.0](https://github.com/qri-io/difff/compare/v0.1.0...v0.2.0) (2020-06-29)


### Bug Fixes

* **calcDeltas:** propagate hasChanges of descendants upward ([12ed525](https://github.com/qri-io/difff/commit/12ed525))
* **diff:** sorting output must respect integer addresses ([b59d728](https://github.com/qri-io/difff/commit/b59d728))
* **int:** Can diff ints, needed for csv body files ([d2c2210](https://github.com/qri-io/difff/commit/d2c2210))
* **stats:** Avoid crashing when array in a map is entirely deleted ([e5f2ec3](https://github.com/qri-io/difff/commit/e5f2ec3))


### Features

* **address:** reperesent address elements as strings, ints, or null ([2610583](https://github.com/qri-io/difff/commit/2610583))
* **changes:** make change calculation optional ([e9dbc5f](https://github.com/qri-io/difff/commit/e9dbc5f))
* **DeepDiff:** move diff functions into methods of a config struct ([6bf7c9a](https://github.com/qri-io/difff/commit/6bf7c9a))


### BREAKING CHANGES

* **DeepDiff:** api for accessing diff methods have moved into methods, added context to request methods



<a name=""></a>
#  (2019-05-30)

This is the first proper release of `deepdiff`. In preparation for go 1.13, in which `go.mod` files and go modules are the primary way to handle go dependencies, we are going to do an official release of all our modules. This will be version v0.1.0 of `deepdiff`.

deepdiff is a structured data differ that aims for near-linear time complexity. It's intended to calculate differences & apply patches to structured data ranging from  0-500MBish of encoded JSON.

Diffing structured data carries additional complexity when compared to the standard unix diff utility, which operates on lines of text. By using the structure of data itself, deepdiff is able to provide a rich description of changes that maps onto the structure of the data itself. deepdiff ignores semantically irrelevant changes like whitespace, and can isolate changes like column changes to tabular data to only the relevant switches


### Bug Fixes

* **optimize:** add extra optimize pass ([613c16a](https://github.com/qri-io/difff/commit/613c16a))
* **walkSorted:** recursion is hard ([e032f79](https://github.com/qri-io/difff/commit/e032f79))


### Features

* **config:** made move calculation configurable, perf work ([9c16ed1](https://github.com/qri-io/difff/commit/9c16ed1))
* **diff:** first signs of life ([fd857c1](https://github.com/qri-io/difff/commit/fd857c1))
* **formatStat:** added stats formatting ([e8366ed](https://github.com/qri-io/difff/commit/e8366ed))
* **moves:** detecting first moves to different parents ([5f38b2a](https://github.com/qri-io/difff/commit/5f38b2a))
* **moves:** we gots moves ([e41c890](https://github.com/qri-io/difff/commit/e41c890))
* **moves:** working on dem moves ([5f7e24b](https://github.com/qri-io/difff/commit/5f7e24b))
* **patch:** support for basic patching ([721e931](https://github.com/qri-io/difff/commit/721e931))
* **stats:** add optional diffstat calculation ([56504e6](https://github.com/qri-io/difff/commit/56504e6))
* first (nonsensical) Deltas showin' up ([fc9f884](https://github.com/qri-io/difff/commit/fc9f884))
* initial work & failing test ([70735a9](https://github.com/qri-io/difff/commit/70735a9))
* use queue to find heavy exact matches ([de01e72](https://github.com/qri-io/difff/commit/de01e72))
* weight-based parent matching propagagion ([d84c363](https://github.com/qri-io/difff/commit/d84c363))
* working on step 2: finding exact matches ([6b38350](https://github.com/qri-io/difff/commit/6b38350))


### Performance Improvements

* **diff:** wow, found the problem thx to our own blog ([25c73aa](https://github.com/qri-io/difff/commit/25c73aa))
* **moves:** parllelize move calculation ([3efa9c2](https://github.com/qri-io/difff/commit/3efa9c2))



