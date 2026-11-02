"""Summarize frozen selection results; validate repeat bytes before reporting."""
import argparse
import json
from pathlib import Path


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("policy", type=Path)
    parser.add_argument("validation", type=Path)
    parser.add_argument("repeat", type=Path)
    args = parser.parse_args()
    for name in ("policy.json", "validation-bindings.json", "validation.json"):
        if (args.validation / name).read_bytes() != (args.repeat / name).read_bytes():
            raise ValueError(f"repeat differs: {name}")
    if args.policy.read_bytes() != (args.validation / "policy.json").read_bytes():
        raise ValueError("frozen policy changed during validation")
    policy = json.loads(args.policy.read_text(encoding="utf-8"))
    rows = json.loads((args.validation / "validation.json").read_text(encoding="utf-8"))
    lines = [
        "# Frozen candidate-selection validation", "",
        "Provisional source-derived labels. Offline experiment, not production selection.", "",
        "Precision here uses actual selected node/file pairs, not a fixed top-k denominator.", "",
        "| Variant | Policy | Development pairs | Development precision | Development recall |",
        "|---|---|---:|---:|---:|",
    ]
    for variant in policy["variants"]:
        trial = next(t for t in variant["development_trials"] if t["policy"] == variant["policy"])
        s = trial["stats"]
        p = variant["policy"]
        lines.append(f"| {variant['variant']} | {p['name']} {p.get('rank_bound', '')} | {s['selected_pairs']} | {s['actual_selected_precision']:.4f} | {s['recall']:.4f} |")
    lines += ["", "## New P2P transfer case", "",
              "| Variant | Before pairs | Selected pairs | Hits | Precision before | Precision selected | Recall selected | Abstained nodes |",
              "|---|---:|---:|---:|---:|---:|---:|---:|"]
    for row in rows:
        b, s = row["baseline"], row["selection"]
        lines.append(f"| {row['variant']} | {b['selected_pairs']} | {s['selected_pairs']} | {s['true_positive_pairs']} | {b['actual_selected_precision']:.4f} | {s['actual_selected_precision']:.4f} | {s['recall']:.4f} | {s['abstained_nodes']} |")
    lines += ["", "## Selected and deferred per node", "",
              "| Variant | Node | Selected | Deferred |", "|---|---|---:|---:|"]
    for row in rows:
        for node in row["nodes"]:
            lines.append(f"| {row['variant']} | {node['element_id']} | {len(node['selected'])} | {len(node['deferred'])} |")
    for row in rows:
        unique = {f for n in row["nodes"] for f in n["selected"]}
        lines += ["", f"{row['variant']}: {len(unique)} distinct selected files across the diagram.", ""]
    lines += [
        "Validation repeats are byte-identical; policy bytes match calibration.", "",
        "The full-context variant keeps all candidates because stricter tested policies lost development hits.",
        "Multi-hop4 uses both branch ranks <=20, selected on earlier Inventory development cases only.",
        "Deferred candidates remain available and are not labelled NO_ACTION. Original ranking/actions stay unchanged.",
        "Six related nodes from one P2P scenario are not six independent validation scenarios.",
        "No confidence probability, guaranteed required-change decision, or generalization claim follows from this result.", "",
    ]
    (args.validation / "selection-summary.md").write_text("\n".join(lines), encoding="utf-8")
    print("repeat artifacts and frozen policy verified; selection-summary.md written")


if __name__ == "__main__":
    main()
