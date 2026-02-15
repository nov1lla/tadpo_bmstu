# Test Run Instructions

## Run all unit tests and generate reports
From the repository root:

```bash
./code/product/run_tests.sh
```

## Run unit → integration → e2e (CI order)
From the repository root:

```bash
./code/product/run_ci_tests.sh
```

## Allure CLI (required for Allure report)
Install the CLI once:

```bash
npm install -g allure-commandline
```

## Output
Reports are generated in:

```
code/product/test-report/
```

Files per module:
- `*-tests.json` — machine-readable test results from `go test -json`
- `allure-results/*.xml` — JUnit XML input for Allure
- `allure-report/` — generated Allure HTML report
- `*-coverage.html` — HTML coverage report
- `*-cover.out` — coverage profile used to build the HTML report

## Integration and E2E tags
- Integration tests run with `-tags integration`
- E2E tests run with `-tags e2e`

## Traffic capture (curl + tcpdump)
From the repository root:

```bash
./code/product/e2e_traffic_capture.sh
```

Artifacts are stored in:
`code/product/traffic/`
