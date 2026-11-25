"""Build experimental pairs from canonical Simple-ERP BPMN files."""

import argparse
from pathlib import Path
import xml.etree.ElementTree as ET

from build_structural_scenarios import diagram, flow, node, q, refresh, tag
from poc_diagram_layout import nfc_layout, redo_layout, skip_layout, switch_layout
from route_active_diagrams import route


def gateway(process, lane, identifier, label, kind='exclusive', direction='Diverging', **attrs):
    node(process, kind + 'Gateway', identifier, label, lane, gatewayDirection=direction, **attrs)


def lane(process, identifier, label):
    return ET.SubElement(process.find(tag('laneSet')), tag('lane'), {'id': identifier, 'name': label})


def automatic_step(process, lane, identifier, label, requirement):
    task = node(process, 'serviceTask', identifier, label, lane)
    ET.SubElement(task, tag('documentation')).text = requirement


def merge_review_outcomes(process, prefix, system, get, validation_failure):
    gateway(process, system, 'DecisionAuditJoin', 'Gabung hasil keputusan', direction='Converging')
    get('Flow_7').set('targetRef', 'DecisionAuditJoin')
    get('Flow_8').set('targetRef', 'DecisionAuditJoin')
    flow(process, 'DecisionReadyForAudit', 'DecisionAuditJoin', prefix + 'effect1')
    gateway(process, system, 'ReviewOutcomeJoin', 'Gabung hasil review', direction='Converging')
    node(process, 'serviceTask', validation_failure, 'Siapkan respons gagal validasi review', system)
    get('Flow_4').set('targetRef', validation_failure)
    get('Flow_3').set('targetRef', 'ReviewOutcomeJoin')
    flow(process, 'ReviewValidationFailureReady', validation_failure, 'ReviewOutcomeJoin')
    flow(process, 'ReturnReviewOutcome', 'ReviewOutcomeJoin', prefix + 'result_activity')


def skip(process, prefix, system, get):
    automatic_step(process, system, 'CheckZeroDelta', 'Periksa apakah semua selisih stok nol',
                   'Saat submit, setelah validasi DRAFT dan versi, bandingkan jumlah terhitung dengan snapshot per baris. '
                   'Satu baris berbeda cukup untuk mengikuti alur SUBMITTED biasa, meski jumlah seluruh selisih nol.')
    gateway(process, system, 'ZeroDeltaDecision', 'Semua selisih nol?')
    automatic_step(process, system, 'CloseWithoutAdjustment', 'Tutup sebagai NO_ADJUSTMENT tanpa posting stok',
                   'Jika semua baris berselisih nol, simpan NO_ADJUSTMENT, naikkan versi, dan audit dalam transaksi submit '
                   'sebelum commit dan respons sukses. Lewati antrean approval; posting stok tetap aksi terpisah. '
                   'Submit dengan versi usang gagal tanpa perubahan.')
    join = get('result_merge')
    if join.get('gatewayDirection') != 'Converging' or get('Flow_5').get('targetRef') != join.get('id'):
        raise ValueError('Skip requires the canonical Adjustment Submit outcome join')
    get('Flow_3').set('targetRef', 'CheckZeroDelta')
    flow(process, 'DeltaChecked', 'CheckZeroDelta', 'ZeroDeltaDecision')
    flow(process, 'BypassApproval', 'ZeroDeltaDecision', 'CloseWithoutAdjustment', 'semua selisih nol')
    flow(process, 'SubmitForApproval', 'ZeroDeltaDecision', prefix + 'effect0', 'ada baris dengan selisih nonnol')
    flow(process, 'BypassCompleted', 'CloseWithoutAdjustment', join.get('id'))


def redo(process, prefix, system, get):
    staff = lane(process, 'RevisionStaff', 'Staff')
    node(process, 'serviceTask', 'LoadFirstReview', 'Muat PR yang menunggu review', system)
    gateway(process, system, 'ReviewEntryJoin', 'Masuk ke review PR', direction='Converging')
    get('Flow_0').set('targetRef', 'LoadFirstReview')
    get('input').set('name', 'Pilih approve, reject, atau request revision')
    automatic_step(process, system, 'ReturnPRForRevision', 'Simpan catatan dan status REVISION_REQUIRED',
                   'Reviewer yang berhak meminta revisi pada PR SUBMITTED dengan versi cocok dan catatan terisi. '
                   'Simpan status, catatan, audit, dan versi baru dalam satu transaksi. '
                   'Kirim respons setelah commit sebelum menunggu perbaikan dari Staff.')
    node(process, 'userTask', 'EditReturnedPR', 'Perbaiki PR yang sama dan ajukan ulang', staff)
    automatic_step(process, system, 'ResubmitRevisedPR', 'Validasi revisi dan simpan PR sebagai SUBMITTED',
                   'Pada request terpisah, pembuat mengirim ulang PR REVISION_REQUIRED dengan versi cocok. '
                   'Simpan revisi dan versi baru pada ID yang sama, audit, commit, lalu kembalikan PR ke antrean review. '
                   'Versi usang atau pengirim yang berbeda gagal tanpa perubahan. REJECTED tetap terminal.')
    flow(process, 'FirstReviewReady', 'LoadFirstReview', 'ReviewEntryJoin')
    flow(process, 'EnterReview', 'ReviewEntryJoin', prefix + 'input')
    revision = flow(process, 'RevisionRequested', prefix + 'valid', 'ReturnPRForRevision',
                    'action = request_revision dan catatan tidak kosong')
    revision.set('name', 'Minta revisi')
    flow(process, 'ReturnedToStaff', 'ReturnPRForRevision', 'EditReturnedPR')
    flow(process, 'ApplyRevision', 'EditReturnedPR', 'ResubmitRevisedPR')
    merge_review_outcomes(process, prefix, system, get, 'RedoValidationFailure')
    loop = flow(process, 'BackToReview', 'ResubmitRevisedPR', 'ReviewEntryJoin')
    loop.set('name', 'Kembali ke antrean review')
    return loop


def switch(process, prefix, system, get):
    get('Flow_5').set('targetRef', 'CheckProductsAtApproval')
    automatic_step(process, system, 'CheckProductsAtApproval', 'Periksa produk di semua baris PR',
                   'Pada aksi approve yang lolos status, versi, hak akses, dan maker-checker, baca status terkini tiap produk. '
                   'Lakukan pemeriksaan dalam transaksi review sebelum mengubah PR. Jika pembacaan gagal, tolak request tanpa commit.')
    gateway(process, system, 'ApprovalProductDecision', 'Ada produk nonaktif?')
    automatic_step(process, system, 'PrepareInactiveReason', 'Siapkan alasan penolakan otomatis',
                   'Satu produk nonaktif cukup untuk menolak PR. Simpan hasil REJECTED dan alasan otomatis pada jalur penolakan yang ada. '
                   'Audit harus mencerminkan hasil penolakan, meski aksi awal approve. Reject manual tetap memakai alasan reviewer.')
    flow(process, 'ProductsChecked', 'CheckProductsAtApproval', 'ApprovalProductDecision')
    flow(process, 'ContinueApproval', 'ApprovalProductDecision', prefix + 'positive', 'semua produk PR aktif')
    flow(process, 'SwitchToRejection', 'ApprovalProductDecision', 'PrepareInactiveReason', 'ada produk PR yang nonaktif')
    gateway(process, system, 'RejectionPathJoin', 'Gabung jalur penolakan', direction='Converging')
    node(process, 'serviceTask', 'KeepManualRejectionReason', 'Gunakan alasan penolakan reviewer', system)
    get('Flow_6').set('targetRef', 'KeepManualRejectionReason')
    flow(process, 'ApplyExistingRejection', 'PrepareInactiveReason', 'RejectionPathJoin')
    flow(process, 'ManualRejectionReady', 'KeepManualRejectionReason', 'RejectionPathJoin')
    flow(process, 'ApplyRejectionOutcome', 'RejectionPathJoin', prefix + 'negative')
    merge_review_outcomes(process, prefix, system, get, 'SwitchValidationFailure')


def stack(process, prefix, system, get):
    second = lane(process, 'SecondApprover', 'Super Admin')
    node(process, 'serviceTask', 'ComputeApprovalTier', 'Hitung jalur approval dari nilai PR', system)
    gateway(process, system, 'ValueDecision', 'PR bernilai tinggi?', default='StandardApproval')
    node(process, 'serviceTask', 'PrepareParallelChecks', 'Siapkan data anggaran dan dokumen PR', system)
    gateway(process, system, 'ChecksSplit', 'Jalankan dua pemeriksaan', kind='parallel')
    gateway(process, system, 'ChecksJoin', 'Kedua pemeriksaan selesai', kind='parallel', direction='Converging')
    node(process, 'serviceTask', 'SummarizeChecks', 'Gabungkan hasil cek anggaran dan dokumen', system)
    gateway(process, system, 'ChecksDecision', 'Kedua cek lolos?', default='ChecksFailed')
    node(process, 'serviceTask', 'QueueSecondTierReview', 'Simpan persetujuan tingkat pertama dan buka antrean Super Admin', system)
    node(process, 'userTask', 'SecondTierReview', 'Review tingkat kedua setelah persetujuan supervisor', second)
    gateway(process, system, 'SecondTierDecision', 'Super Admin menyetujui?', default='SecondTierRejected')
    node(process, 'serviceTask', 'PrepareChecksFailure', 'Siapkan hasil gagal dan pertahankan status PR', system)
    get('Flow_5').set('targetRef', 'ComputeApprovalTier')
    flow(process, 'TierComputed', 'ComputeApprovalTier', 'ValueDecision')
    flow(process, 'StandardApproval', 'ValueDecision', prefix + 'positive')
    flow(process, 'HighValueApproval', 'ValueDecision', 'PrepareParallelChecks', 'nilai PR mencapai ambang approval tingkat kedua')
    flow(process, 'CheckInputsReady', 'PrepareParallelChecks', 'ChecksSplit')
    for identifier, label in [('CheckBudget', 'Periksa kecukupan anggaran PR'), ('CheckDocuments', 'Periksa kelengkapan dokumen PR')]:
        node(process, 'serviceTask', identifier, label, system)
        flow(process, identifier + 'Start', 'ChecksSplit', identifier)
        flow(process, identifier + 'Done', identifier, 'ChecksJoin')
    flow(process, 'AggregateChecks', 'ChecksJoin', 'SummarizeChecks')
    flow(process, 'DecideChecks', 'SummarizeChecks', 'ChecksDecision')
    flow(process, 'BothPassed', 'ChecksDecision', 'QueueSecondTierReview', 'anggaran cukup dan dokumen lengkap')
    flow(process, 'SecondTierQueued', 'QueueSecondTierReview', 'SecondTierReview')
    flow(process, 'ChecksFailed', 'ChecksDecision', 'PrepareChecksFailure')
    flow(process, 'ReturnChecksFailure', 'PrepareChecksFailure', prefix + 'result_activity')
    flow(process, 'SecondTierChoice', 'SecondTierReview', 'SecondTierDecision')
    flow(process, 'SecondTierApproved', 'SecondTierDecision', prefix + 'positive', 'Super Admin menyetujui')
    flow(process, 'SecondTierRejected', 'SecondTierDecision', prefix + 'negative')


def non_free_choice(process, prefix, system, get):
    reviewer = lane(process, 'AssignedReviewer', 'Reviewer')
    automatic_step(process, system, 'LockApprovalRoute', 'Tentukan dan simpan jalur approval PR',
                   'Saat submit, total PR di bawah ambang konfigurasi mendapat STANDARD; total yang mencapai ambang mendapat ESCALATED. '
                   'Simpan rute untuk ID PR yang sama bersama status SUBMITTED dalam satu transaksi. '
                   'Perubahan ambang berikutnya hanya berlaku pada submit baru.')
    node(process, 'sendTask', 'ReturnSubmitReceipt', 'Kirim konfirmasi submit ke Staff', system)
    request = node(process, 'userTask', 'RequestLockedReview', 'Kirim keputusan approve atau reject', reviewer)
    ET.SubElement(request, tag('documentation')).text = (
        'Ini request review PR yang sudah ada dan dapat datang setelah transaksi submit selesai.'
    )
    automatic_step(process, system, 'LoadLockedApprovalRoute', 'Baca rute yang tersimpan untuk PR ini',
                   'Pada request approve maupun reject, baca rute PR yang sama sebelum mutasi. '
                   'Perubahan ambang tidak mengubah rute PR yang sudah SUBMITTED. '
                   'Pertahankan pemeriksaan status, versi, kewenangan, dan maker-checker.')
    gateway(process, system, 'LockedRouteDecision', 'Rute review yang tersimpan?')
    gateway(process, system, 'LockedReviewJoin', 'Gabung hasil rute', direction='Converging')
    gateway(process, system, 'NFCOutcomeJoin', 'Gabung hasil proses', direction='Converging')
    node(process, 'serviceTask', 'PrepareLockedReviewOutcome', 'Siapkan hasil keputusan review', system)
    node(process, 'serviceTask', 'NFCSubmitValidationFailure', 'Siapkan respons gagal submit', system)
    get('Flow_6').set('targetRef', 'NFCSubmitValidationFailure')
    flow(process, 'NFCSubmitFailureReady', 'NFCSubmitValidationFailure', 'NFCOutcomeJoin')
    flow(process, 'PrepareLockedOutcome', 'LockedReviewJoin', 'PrepareLockedReviewOutcome')
    flow(process, 'LockedOutcomeReady', 'PrepareLockedReviewOutcome', 'NFCOutcomeJoin')
    flow(process, 'ReturnNFCOutcome', 'NFCOutcomeJoin', prefix + 'result_activity')
    for identifier, label, requirement in [
        ('OpenStandardReview', 'Validasi Supervisor dan lanjutkan review',
         'STANDARD memberi hak review kepada Supervisor. Tolak role yang tidak cocok sebelum mutasi. '
         'Untuk role yang cocok, lanjutkan aksi approve atau reject yang sudah ada.'),
        ('OpenEscalatedReview', 'Validasi Super Admin dan lanjutkan review',
         'ESCALATED memberi hak review kepada Super Admin. Tolak role yang tidak cocok sebelum mutasi. '
         'Untuk role yang cocok, lanjutkan aksi approve atau reject yang sudah ada.'),
        ('RejectUnknownRoute', 'Tolak review tanpa mutasi',
         'PR yang tidak punya rute valid gagal secara eksplisit tanpa mengubah status atau versi.'),
    ]:
        automatic_step(process, system, identifier, label, requirement)
        flow(process, identifier + 'Done', identifier, 'LockedReviewJoin')
    get('Flow_3').set('targetRef', 'LockApprovalRoute')
    get('Flow_5').set('targetRef', 'ReturnSubmitReceipt')
    get('Flow_5').attrib.pop('name', None)
    flow(process, 'RouteLocked', 'LockApprovalRoute', prefix + 'effect0')
    continue_review = flow(process, 'ContinueToReview', 'ReturnSubmitReceipt', 'RequestLockedReview')
    continue_review.set('name', 'Request review berikutnya')
    flow(process, 'ReadLockedRoute', 'RequestLockedReview', 'LoadLockedApprovalRoute')
    flow(process, 'RouteLoaded', 'LoadLockedApprovalRoute', 'LockedRouteDecision')
    flow(process, 'StoredStandard', 'LockedRouteDecision', 'OpenStandardReview', 'jalur tersimpan = STANDARD')
    flow(process, 'StoredEscalated', 'LockedRouteDecision', 'OpenEscalatedReview', 'jalur tersimpan = ESCALATED')
    missing = flow(process, 'MissingRoute', 'LockedRouteDecision', 'RejectUnknownRoute',
                   'rute kosong atau di luar STANDARD dan ESCALATED')
    missing.set('name', 'Rute kosong atau tidak valid')


def skip_evaluation_baseline(root, process, elements, prefix):
    """Keep the canonical graph and describe only observable behavior."""
    process.find(tag('documentation')).text = (
        'Level 3. Staff mengirim penyesuaian stok untuk ditinjau. '
        'Sistem memeriksa status dan versi sebelum memproses permintaan.'
    )
    conflict = elements[prefix + 'Flow_6_prepare_result']
    conflict.find(tag('documentation')).text = (
        'Jika versi tidak cocok atau status bukan DRAFT, siapkan respons konflik 409. '
        'Status penyesuaian stok tetap sama.'
    )
    return ET.tostring(root, encoding='utf-8', xml_declaration=True)


def build(repo, output, scenario=None):
    sources = {
        'skip': ('staff/inventory/level-3/role-staff-l3-adj-submit.bpmn', skip),
        'redo': ('supervisor/p2p/level-3/role-supervisor-l3-pr-review.bpmn', redo),
        'switch': ('supervisor/p2p/level-3/role-supervisor-l3-pr-review.bpmn', switch),
        'xor-and-stack': ('supervisor/p2p/level-3/role-supervisor-l3-pr-review.bpmn', stack),
        'non-free-choice': ('staff/p2p/level-3/role-staff-l3-pr-submit.bpmn', non_free_choice),
    }
    for name, (relative, change) in sources.items():
        if scenario is not None and scenario != name:
            continue
        data = (repo / 'docs/bpm-haqi/roles' / relative).read_bytes()
        root = ET.fromstring(data)
        process = root.find(tag('process'))
        system = process.findall('.//' + tag('lane'))[1]
        prefix = Path(relative).stem.replace('-', '_') + '_'
        elements = {element.get('id'): element for element in process}
        if name == 'skip':
            data = skip_evaluation_baseline(root, process, elements, prefix)
        loop = change(process, prefix, system, lambda suffix: elements[prefix + suffix])
        # Lay out the forward graph first, then route the loop around its shapes.
        if loop is not None:
            process.remove(loop)
        diagram(root)
        if loop is not None:
            process.append(loop)
            refresh(process)
            plane = root.find('.//' + q('bpmndi', 'BPMNPlane'))
            identifier = loop.get('id')
            ET.SubElement(plane, q('bpmndi', 'BPMNEdge'), {'id': identifier + '_di', 'bpmnElement': identifier})
        layout = {'skip': skip_layout, 'redo': redo_layout, 'switch': switch_layout, 'non-free-choice': nfc_layout}.get(name)
        if layout:
            layout(root, prefix)
        else:
            route(root)
        ET.indent(root)
        folder = output / name
        folder.mkdir(parents=True, exist_ok=True)
        (folder / 'before.bpmn').write_bytes(data)
        ET.ElementTree(root).write(folder / 'after.bpmn', encoding='utf-8', xml_declaration=True)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--repo', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--scenario', choices=['skip', 'redo', 'switch', 'non-free-choice', 'xor-and-stack'])
    args = parser.parse_args()
    build(args.repo, args.output, args.scenario)
