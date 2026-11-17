# Tools penelitian

Script pendukung dipisahkan dari input BPMN aktif. Jalankan dari root mapper:

```powershell
python -m unittest discover -s experiments/tools -p "test_*.py"
python experiments/tools/verify_current_staff_pack.py --help
```

Script bukan engine produksi. Jangan memasukkan acuan evaluasi ke inference. Source Simple-ERP tetap read-only. Paket aktif berada di experiments/simple-erp/current-staff-v2.
