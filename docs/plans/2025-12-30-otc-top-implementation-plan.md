# OTC Top Implementation Plan

## Scope

Implement a new coin source entry `otc_top` using the OTC Top API:
`http://168.138.207.11:3080/api/public/top-otc-crypto`.

## Tasks

1. Update strategy config schema to include OTC Top fields and source type.
2. Implement OTC Top provider parsing and sorting by `otc_index`.
3. Wire `otc_top` into candidate selection and backtest/debate resolution.
4. Update frontend types, Strategy Studio UI, and i18n strings.
5. Update docs/tests for OTC Top behavior.
