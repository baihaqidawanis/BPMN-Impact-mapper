param([Parameter(Mandatory=$true)][string]$OutputDirectory)
$ErrorActionPreference = 'Stop'
# Post-inference reporting only: no source repository or reference inputs.
$benchmark = Get-Content -LiteralPath (Join-Path $OutputDirectory 'benchmark-report.json') -Raw | ConvertFrom-Json
$rows = foreach ($result in $benchmark.results) {
    $folder = Join-Path (Join-Path $OutputDirectory $result.scenario_id) $result.method
    $predictionPath = Join-Path $folder 'impact-report.json'
    if ((Get-FileHash -LiteralPath $predictionPath -Algorithm SHA256).Hash.ToLowerInvariant() -ne $result.prediction_sha256) {
        throw "Prediction hash mismatch: $($result.scenario_id)/$($result.method)"
    }
    $evaluation = Get-Content -LiteralPath (Join-Path $folder 'evaluation-report.json') -Raw | ConvertFrom-Json
    if ($evaluation.evaluation_id -ne $result.evaluation_id) { throw 'Evaluation identity mismatch' }
    $prediction = Get-Content -LiteralPath $predictionPath -Raw | ConvertFrom-Json
    foreach ($change in $evaluation.changes) {
        $predicted = @($change.predicted_files)
        $expected = @($change.expected_files)
        $raw = @($prediction.changes | Where-Object {
            $id = if ($_.new_element) { $_.new_element.id } else { $_.old_element.id }
            $_.change_type -eq $change.change_type -and $id -eq $change.element_id
        })
        if ($raw.Count -ne 1) { throw 'Evaluation change has no unique prediction' }
        $kind = if ($raw[0].part_of) { 'supporting-flow' } elseif ($expected.Count -eq 0) { 'independent-empty-reference' } else { 'positive-reference' }
        $hits5 = @($predicted | Select-Object -First 5 | Where-Object { $expected -contains $_ }).Count
        $hits10 = @($predicted | Select-Object -First 10 | Where-Object { $expected -contains $_ }).Count
        [pscustomobject][ordered]@{
            benchmark_id = $benchmark.benchmark_id; label_provenance = $benchmark.label_provenance
            input_fingerprint = $result.input_fingerprint
            scenario = $result.scenario_id; method = $result.method
            change_type = $change.change_type; element_id = $change.element_id
            evaluation_group = $kind; expected_count = $expected.Count; predicted_count = $predicted.Count
            precision_at_5 = if ($expected.Count) { $hits5 / 5.0 } else { $null }
            recall_at_5 = if ($expected.Count) { $hits5 / [double]$expected.Count } else { $null }
            precision_at_10 = if ($expected.Count) { $hits10 / 10.0 } else { $null }
            recall_at_10 = if ($expected.Count) { $hits10 / [double]$expected.Count } else { $null }
            reciprocal_rank = if ($expected.Count) { $change.reciprocal_rank } else { $null }
            average_precision = if ($expected.Count) { $change.average_precision } else { $null }
            r_precision = if ($expected.Count) { @($predicted | Select-Object -First $expected.Count | Where-Object { $expected -contains $_ }).Count / [double]$expected.Count } else { $null }
            expected_ranks = (($expected | ForEach-Object { "$_=" + ([array]::IndexOf($predicted, $_) + 1) }) -join '; ')
            missed_files = (($expected | Where-Object { $predicted -notcontains $_ }) -join '; ')
            extra_files = (($predicted | Where-Object { $expected -notcontains $_ }) -join '; ')
            reproducible = $result.reproducible
        }
    }
}
$rows | Export-Csv -LiteralPath (Join-Path $OutputDirectory 'change-diagnostics.csv') -NoTypeInformation -Encoding UTF8
$aggregates = foreach ($method in ($benchmark.results.method | Sort-Object -Unique)) {
    $positive = @($rows | Where-Object { $_.method -eq $method -and $_.evaluation_group -eq 'positive-reference' })
    $controls = @($rows | Where-Object { $_.method -eq $method -and $_.evaluation_group -eq 'independent-empty-reference' })
    [pscustomobject][ordered]@{
        benchmark_id = $benchmark.benchmark_id; label_provenance = $benchmark.label_provenance
        method = $method; positive_changes = $positive.Count
        precision_at_5 = ($positive | Measure-Object precision_at_5 -Average).Average
        recall_at_5 = ($positive | Measure-Object recall_at_5 -Average).Average
        precision_at_10 = ($positive | Measure-Object precision_at_10 -Average).Average
        recall_at_10 = ($positive | Measure-Object recall_at_10 -Average).Average
        mrr = ($positive | Measure-Object reciprocal_rank -Average).Average
        map = ($positive | Measure-Object average_precision -Average).Average
        r_precision = ($positive | Measure-Object r_precision -Average).Average
        independent_empty_cases = $controls.Count
        correct_abstentions = @($controls | Where-Object predicted_count -eq 0).Count
    }
}
$aggregates | Export-Csv -LiteralPath (Join-Path $OutputDirectory 'positive-change-summary.csv') -NoTypeInformation -Encoding UTF8
$aggregates | Format-Table -AutoSize
