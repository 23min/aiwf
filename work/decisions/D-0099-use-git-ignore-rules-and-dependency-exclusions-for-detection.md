---
id: D-0099
title: Use Git ignore rules and dependency exclusions for detection
status: accepted
relates_to:
    - M-0347
    - E-0094
---
> **Date:** 2026-09-22 · **Decided by:** Peter Bruinsma

## Question

Which project files should contribute to guidance suggestions when a directory name can describe either generated output or handwritten code?

## Decision

Respect Git ignore rules and explicitly skip dependency/cache directories. Do not exclude directories solely because they have ambiguous output names such as `bin`, `build`, or `out`. Keep nested repositories and symbolic links outside the scan.

## Reasoning

The legacy detector's broad directory-name exclusions can hide handwritten project tools. Git ignore rules express project-specific exclusions without adding a new configuration mechanism. Explicit dependency/cache exclusions avoid suggestions based on third-party material even where that material is not ignored. Parsing file contents to identify generated code would add language-specific maintenance to aiwf.

## Consequences

Projects must use Git ignore rules to exclude generated material outside the dependency/cache exclusions. Detection does not claim to recognize every generated file. M-0347 tests the exclusion boundary separately from external language patterns.
