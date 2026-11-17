"""Source-reviewed case-study reference. No retrieval, repository execution or LLM.

Explicit review specifications choose functions/lines; text matching only anchors
those decisions. It does not discover semantic mappings automatically.
"""
import argparse
import hashlib
import html
import json
import re
import subprocess
from pathlib import Path
import xml.etree.ElementTree as ET

NS={'b':'http://www.omg.org/spec/BPMN/20100524/MODEL'}
ROOT='docs/bpm-haqi/roles/staff/p2p/level-3'


def sha(data): return hashlib.sha256(data).hexdigest()
def encoded(value): return (json.dumps(value,ensure_ascii=False,indent=2)+'\n').encode('utf-8')
def module_path(module,part):return f'backend/internal/{module}/{part}'

# Each choice below comes from static source review, not mapper candidates.
SPECS={
 'pr-create':dict(module='purchasing',store='CreateRequest',handler='create',ui='purchase-request-panel',component='PurchaseRequestPanel',input=['onSubmit={create}','Tujuan pembelian'],check=['len(lines) == 0','l.Quantity <= 0','FROM products'],effects=[['INSERT INTO purchase_requests','INSERT INTO purchase_request_lines'],['"PUR-PR-001"','tx.Commit(']]),
 'pr-submit':dict(module='purchasing',store='TransitionRequest',handler='prAction',ui='purchase-request-panel',component='PurchaseRequestPanel',input=['action(document, "submit")'],check=['FOR UPDATE','d.Version != version','case "submit":','from != "DRAFT"'],effects=[['d.Status = "SUBMITTED"','UPDATE purchase_requests'],['trace := "PUR-PR-002"','audit(ctx','tx.Commit(']]),
 'pr-cancel':dict(module='purchasing',store='TransitionRequest',handler='prAction',ui='purchase-request-panel',component='PurchaseRequestPanel',input=['const reason =','action(document, "cancel")'],check=['d.Version != version','role=\'SUPER_ADMIN\'','from != "DRAFT" || reason == ""'],effects=[['d.Status = "CANCELLED"','UPDATE purchase_requests'],['"ACT-PUR-PR-CANCEL"','audit(ctx','tx.Commit(']]),
 'pr-copy':dict(module='purchasing',store='CopyRequest',handler='prAction',ui='purchase-request-panel',component='PurchaseRequestPanel',input=['action(document, "copy")'],check=['NOT EXISTS(','!mastersActive'],effects=[['copied_from_id','INSERT INTO purchase_request_lines'],['"ACT-PUR-PR-COPY"','tx.Commit(']]),
 'po-convert':dict(module='purchasing',store='ConvertToOrder',handler='prAction',ui='purchase-order-panel',component='PurchaseOrderPanel',input=['function convert(','resource="suppliers"','resource="warehouses"'],check=['status != "APPROVED"','FROM suppliers','FROM warehouses'],effects=[['INSERT INTO purchase_orders','INSERT INTO purchase_order_lines'],["status='CONVERTED_TO_PO'",'"PUR-PO-001"','tx.Commit(']]),
 'gr-create':dict(module='purchasing',store='CreateReceipt',handler='receipt',ui='goods-receipt-panel',component='GoodsReceiptPanel',input=['function create(','PO Line'],check=['status != "ISSUED"','l.Quantity <= 0','FROM purchase_order_lines','!valid'],effects=[['INSERT INTO goods_receipts','INSERT INTO goods_receipt_lines'],['"PUR-GR-001"','tx.Commit(']]),
 'gr-cancel':dict(module='purchasing',store='CancelReceipt',handler='cancelReceipt',ui='goods-receipt-panel',component='GoodsReceiptPanel',input=['Alasan reverse/cancel','/cancel`'],check=['reason == ""',"status='DRAFT'"],effects=[["status='CANCELLED'",'version=version+1'],['"ACT-PUR-GR-CANCEL"','tx.Commit(']]),
 'ap-invoice':dict(module='finance',store='CreateSupplierInvoice',handler='createSupplierInvoice',ui='finance-workspace',component='FinancePanel',input=['function createInvoice(','command("/supplier-invoices"'],check=['len(input.Lines) == 0','status != "RECEIVED"','!replacementValid','line.Quantity <= 0'],effects=[['INSERT INTO supplier_invoices',"'MATCH_PENDING'",'INSERT INTO supplier_invoice_lines'],['"PUR-INV-001"','tx.Commit(']]),
 'ap-resolve':dict(module='finance',store='ResolveSupplierInvoice',handler='resolveSupplierInvoice',ui='finance-workspace',component='FinancePanel',input=['function beginResolve(','/resolve-match`'],check=['len(lines) == 0','document.Status != "MATCH_EXCEPTION"','line.Quantity <= 0'],effects=[['DELETE FROM supplier_invoice_lines','INSERT INTO supplier_invoice_lines','total=$2,outstanding=$2'],["status='MATCH_PENDING'",'"PUR-INV-003"','tx.Commit(']]),
 'ap-payment':dict(module='finance',store='CreateSupplierPayment',handler='createSupplierPayment',ui='finance-workspace',component='FinancePanel',input=['function createPayment(','allocations: [{ invoiceId: invoice'],check=['allocationTotal(','amount != total','FROM suppliers'],effects=[['INSERT INTO supplier_payments',"'DRAFT'",'INSERT INTO supplier_payment_allocations'],['"PUR-PAY-001"','tx.Commit(']]),
}


def build(repo):
    hashes={};cache={}
    def lines(file):
        if file not in cache:
            data=(repo/file).read_bytes();hashes[file]=sha(data);cache[file]=data.decode('utf-8-sig').splitlines()
        return cache[file]
    def relation(file,function,role,needles):
        content=lines(file)
        if file.endswith('.go'):
            name=function.split('.')[-1]
            starts=[i for i,l in enumerate(content) if l.startswith('func ')]
            hits=[i for i in starts if re.search(r'\b'+re.escape(name)+r'\(',content[i])]
            if len(hits)!=1:raise ValueError(('Function anchor ambiguous',file,function,hits))
            start=hits[0];end=next((i for i in starts if i>start),len(content))
        else:
            start=next(i for i,l in enumerate(content) if re.search(r'function\s+'+function+r'\b',l));end=len(content)
        evidence=[]
        for needle in needles:
            matches=[i for i in range(start,end) if needle in content[i]]
            if not matches:raise ValueError(('Reviewed evidence no longer matches',file,function,needle))
            i=matches[0]
            if any(e['line']==i+1 for e in evidence):continue
            evidence.append({'line':i+1,'excerpt':content[i].strip()[:320],'match':needle})
        return {'file':file,'function':function,'role':role,'function_start_line':start+1,'function_end_line':end,'evidence':evidence}
    processes=[];all_activity_ids=[]
    for suffix,spec in sorted(SPECS.items()):
        file=f'{ROOT}/role-staff-l3-{suffix}.bpmn';data=(repo/file).read_bytes();hashes[file]=sha(data)
        p=ET.fromstring(data).find('b:process',NS)
        store=module_path(spec['module'],'adapter/postgres/store.go');handler=module_path(spec['module'],'adapter/http/handler.go');service=module_path(spec['module'],'application/service.go')
        frontend_module='finance' if spec['module']=='finance' else 'purchasing'
        frontend=f'frontend/features/{frontend_module}/{spec["ui"]}.tsx'
        actors={ref.text:lane.get('name') for lane in p.findall('.//b:lane',NS) for ref in lane.findall('b:flowNodeRef',NS)}
        activities=[]
        for e in p:
            kind=e.tag.split('}')[-1]
            if kind not in ['userTask','serviceTask','sendTask']:continue
            id=e.get('id');all_activity_ids.append(id)
            if id.endswith('_input'):links=[relation(frontend,spec['component'],'USER_INPUT',spec['input'])]
            elif id.endswith('_check'):links=[relation(store,'Store.'+spec['store'],'VALIDATION',spec['check'])]
            elif '_effect' in id:
                index=int(id.rsplit('effect',1)[1]);links=[relation(store,'Store.'+spec['store'],'PERSISTENCE' if index==0 else 'AUDIT_AND_COMMIT',spec['effects'][index])]
            elif id.endswith('_result_activity'):
                if spec['module']=='finance':
                    links=[relation(handler,'Handler.'+spec['handler'],'RESPONSE_DISPATCH',['result(w, document, err,']),relation(handler,'result','SUCCESS_ERROR_RESPONSE',['if err != nil','respond(w, status, document)']),relation(handler,'respond','HTTP_JSON_RESPONSE',['w.WriteHeader(status)','json.NewEncoder(w)'])]
                else:
                    links=[relation(handler,'Handler.'+spec['handler'],'SUCCESS_ERROR_RESPONSE',['respond(w, 409','respond(w, 20']),relation(handler,'respond','HTTP_JSON_RESPONSE',['w.WriteHeader(s)','json.NewEncoder(w)'])]
            else:raise ValueError(('Unreviewed activity',id))
            activities.append({'task_id':id,'name':e.get('name'),'type':kind,'actor':actors[id],'review_status':'provisional-source-reviewed','mappings':links})
        # Contextual call chain is not duplicated as a direct implementation of
        # every task, and is not treated as a required-change label.
        context=[relation(handler,'Handler.'+spec['handler'],'API_SERVICE_BINDING',['h.Service.'+spec['store']+'(']),
                 relation(service,'Service.'+spec['store'],'DELEGATION',['s.store.'+spec['store']+'('])]
        if suffix=='ap-invoice':
            context += [relation(store,'supplierReplacementValid','VALIDATION_HELPER',['return validateReplacement(']),relation(store,'validateReplacement','REPLACEMENT_RULE',['latestStatus'])]
        if suffix=='ap-payment':context.append(relation(store,'allocationTotal','VALIDATION_HELPER',['allocations','return']))
        if suffix in ['pr-create','pr-submit','po-convert','gr-create','ap-invoice','ap-payment']:
            context.append(relation('backend/test/integration/operations_test.go','TestP2PPartialReceiptPostsImmutableSourceLinkedLedger','RELATED_TEST_NOT_EXECUTED',[spec['store']+'(']))
        elif suffix=='pr-copy':
            context.append(relation('backend/test/integration/operations_test.go','TestCopyRejectedDocumentsRequiresActiveMasters','RELATED_TEST_NOT_EXECUTED',['purchasing.CopyRequest(']))
        elif suffix=='ap-resolve':
            context.append(relation('backend/test/integration/finance_test.go','TestP2PConsolidatedInvoiceExactMatchPartialPaymentCloseAndVoid','RELATED_TEST_NOT_EXECUTED',['finance.ResolveSupplierInvoice(']))
        controls=[{'id':e.get('id'),'type':e.tag.split('}')[-1],'name':e.get('name',''),'status':'control-flow element; not assigned a standalone file mapping'} for e in p if e.tag.split('}')[-1].endswith(('Event','Gateway'))]
        processes.append({'bpmn_file':file,'process_id':p.get('id'),'name':p.get('name'),'activities':activities,'operation_context':context,'control_elements':controls})
    assert len(processes)==10 and len(all_activity_ids)==50 and len(set(all_activity_ids))==50
    revision=subprocess.check_output(['git','-C',str(repo),'rev-parse','HEAD'],text=True).strip()
    dirty=bool(subprocess.check_output(['git','-C',str(repo),'status','--porcelain'],text=True))
    result={'schema_version':'1','purpose':'research-only baseline traceability; excluded from inference',
        'review_status':'provisional single AI source review; no independent review or runtime proof',
        'repository':{'name':repo.name,'git_head':revision,'working_tree_dirty':dirty},
        'scope':{'role':'Staff','module':'P2P','level':3,'processes':10,'activities':50},
        'limitations':['Existing implementation relationships do not imply required code changes.','No test coverage completeness or function-level mapping accuracy measured.','Other roles/modules, gateway branch semantics and shared domain/schema dependencies are not exhaustively mapped.','Function bounds in this research script use declaration text, not a claimed TypeScript AST adapter.'],
        'file_sha256':dict(sorted(hashes.items())),'processes':processes}
    result['snapshot_id']=sha(encoded(result))[:32]
    return result


def render(value):
    def esc(s):return html.escape(str(s),quote=False).replace('|','\\|').replace('\n',' ')
    roles={'USER_INPUT':'Input pengguna','VALIDATION':'Validasi aturan','PERSISTENCE':'Penyimpanan data/state','AUDIT_AND_COMMIT':'Audit dan commit','RESPONSE_DISPATCH':'Delegasi hasil respons','SUCCESS_ERROR_RESPONSE':'Respons sukses/gagal','HTTP_JSON_RESPONSE':'Pengiriman JSON HTTP'}
    out=['# Mapping baseline Staff P2P L3','',f'Snapshot: `{value["snapshot_id"]}`. Git HEAD: `{value["repository"]["git_head"]}`; dirty: {value["repository"]["working_tree_dirty"]}.',
         '', '10 proses, 50 aktivitas. Acuan sementara dari inspeksi source oleh satu reviewer AI; belum review independen atau bukti runtime.',
         'Tabel menunjukkan implementasi existing. Hubungan ini bukan daftar file yang otomatis wajib berubah. Tidak digunakan sebagai input retrieval.',
         '', 'JSON adalah data utama; Markdown ini dirender dari object JSON yang sama. Seluruh path relatif terhadap root Simple-ERP. Nomor baris berlaku pada hash snapshot.', '']
    for process in value['processes']:
        out+=['## '+esc(process['name']),'',f'BPMN: `{process["bpmn_file"]}`','', '| Task ID / nama | File | Function | Peran | Bukti baris kode |','|---|---|---|---|---|']
        for a in process['activities']:
            for link in a['mappings']:
                evidence='<br>'.join(f'L{e["line"]}: {esc(e["excerpt"])}' for e in link['evidence'])
                out.append('| '+ ' | '.join([esc(a['task_id'])+'<br>'+esc(a['name'])+'<br>'+esc(a['type']+' / '+a['actor']),esc(link['file']),esc(link['function']),roles[link['role']],evidence])+' |')
        out+=['','Rantai operasi pendukung (bukan direct mapping semua task):','']
        for link in process['operation_context']:
            out.append(f'- `{link["file"]}` / `{link["function"]}` — {link["role"]}: '+ '; '.join(f'L{e["line"]}: {esc(e["excerpt"])}' for e in link['evidence']))
        out+=['',f'{len(process["control_elements"])} event/gateway dicatat di JSON sebagai elemen kontrol; tidak diberi file tersendiri secara otomatis.','']
    out+=['## Batas penggunaan','']+['- '+s for s in value['limitations']]+['']
    return ('\n'.join(out)).encode('utf-8')


if __name__=='__main__':
    parser=argparse.ArgumentParser();parser.add_argument('--repo',type=Path,required=True);parser.add_argument('--output',type=Path,required=True);parser.add_argument('--verify',action='store_true');args=parser.parse_args()
    value=build(args.repo);data=encoded(value);markdown=render(json.loads(data))
    if args.verify:
        assert (args.output/'staff-p2p-l3.json').read_bytes()==data,'Snapshot drift; inspect source/BPMN before regenerating'
        assert (args.output/'staff-p2p-l3.md').read_bytes()==markdown,'Markdown not rendered from canonical JSON'
        print('Verified: ten processes, all fifty activities, source evidence, hashes, canonical JSON and Markdown match.')
    else:
        args.output.mkdir(parents=True,exist_ok=False)
        (args.output/'staff-p2p-l3.json').write_bytes(data);(args.output/'staff-p2p-l3.md').write_bytes(markdown)
        print('Created baseline snapshot',value['snapshot_id'])
