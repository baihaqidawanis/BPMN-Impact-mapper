"""Freeze a P2P transfer check after locking multi-hop4, before new inference."""
import argparse
import copy
import json
from pathlib import Path
import xml.etree.ElementTree as ET
from build_structural_scenarios import tag,node,flow,diagram,sha


def write_json(path,value):path.write_bytes((json.dumps(value,indent=2)+'\n').encode())


def build(repo,out):
    out.mkdir(parents=True,exist_ok=False)
    source='docs/bpm-haqi/roles/staff/p2p/level-3/role-staff-l3-pr-submit.bpmn'
    data=(repo/source).read_bytes();base=ET.fromstring(data);prefix='role_staff_l3_pr_submit_'
    store='backend/internal/purchasing/adapter/postgres/store.go';tests='backend/test/integration/operations_test.go'
    files=[store,tests,'backend/internal/purchasing/adapter/http/handler.go','frontend/features/purchasing/purchase-request-panel.tsx']
    labels=[('SubmitChecksSplit','ADD_GATEWAY'),('CheckActiveProducts','ADD_TASK'),('CheckEstimatedPrices','ADD_TASK'),('SubmitChecksJoin','ADD_GATEWAY'),('CombineSubmitChecks','ADD_TASK'),('SubmitEligibility','ADD_GATEWAY')]
    scenarios=[]
    for name in ['checks','unchanged-control']:
        folder=out/name;folder.mkdir();(folder/'before.bpmn').write_bytes(data);root=copy.deepcopy(base)
        if name=='checks':
            p=root.find(tag('process'));system=next(l for l in p.findall('.//'+tag('lane')) if l.get('name')=='Sistem Simple ERP')
            bridge=next(f for f in p.findall(tag('sequenceFlow')) if f.get('sourceRef')==prefix+'input' and f.get('targetRef')==prefix+'check');bridge.set('targetRef','SubmitChecksSplit')
            node(p,'parallelGateway','SubmitChecksSplit','Run both purchase request submission checks',system,gatewayDirection='Diverging')
            node(p,'serviceTask','CheckActiveProducts','Recheck all purchase request products are active',system)
            node(p,'serviceTask','CheckEstimatedPrices','Check every purchase request estimated price is positive',system)
            node(p,'parallelGateway','SubmitChecksJoin','Collect both purchase request validation results',system,gatewayDirection='Converging')
            node(p,'serviceTask','CombineSubmitChecks','Aggregate purchase request product and price checks without changing state',system)
            node(p,'exclusiveGateway','SubmitEligibility','Both purchase request checks passed?',system,gatewayDirection='Diverging',default='SubmitChecksFailed')
            for id in ['CheckActiveProducts','CheckEstimatedPrices']:
                flow(p,id+'Start','SubmitChecksSplit',id);flow(p,id+'Done',id,'SubmitChecksJoin')
            flow(p,'CombineChecks','SubmitChecksJoin','CombineSubmitChecks');flow(p,'ChooseOutcome','CombineSubmitChecks','SubmitEligibility')
            flow(p,'SubmitChecksPassed','SubmitEligibility',prefix+'check','all products active and all estimated prices > 0')
            rejected=flow(p,'SubmitChecksFailed','SubmitEligibility',prefix+'result_merge');rejected.set('name','Inactive product or non-positive estimated price: reject without submit')
            diagram(root);ET.indent(root);after=ET.tostring(root,encoding='utf-8',xml_declaration=True)
        else:after=data
        (folder/'after.bpmn').write_bytes(after)
        truth={'version':'1','cases':[{'change_type':kind,'element_id':id,'expected':[{'file':f,'required_action':'MODIFY'} for f in [store,tests]]} for id,kind in labels] if name=='checks' else []}
        write_json(folder/'reference.json',truth)
        scenarios.append({'id':'p2p-transfer-'+name,'repository_path':'../Simple-ERP','repository_name':'Simple-ERP','revision':'p2p-transfer-frozen',
            'old_bpmn_path':(folder/'before.bpmn').as_posix(),'old_bpmn_sha256':sha(data),'new_bpmn_path':(folder/'after.bpmn').as_posix(),'new_bpmn_sha256':sha(after),
            'ground_truth_path':(folder/'reference.json').as_posix(),'ground_truth_sha256':sha((folder/'reference.json').read_bytes()),'truth_scope':'subset' if name=='checks' else 'complete'})
    write_json(out/'source-review.json',{'requirement':'At PR submit, recheck all linked products active and every estimated price strictly positive. Collect both results; any failure leaves DRAFT and does not commit submission. Retain actor/version/state guards.',
        'reference_status':'provisional single-AI source review; new P2P case withheld from initial inventory context development, not independent ground truth',
        'policy_lock':'multi-hop4 depth=4 visited=128 anchors=2/direction, no changes after this fixture is scored',
        'source_hashes':{f:sha((repo/f).read_bytes()) for f in files},'baseline_source':source,'baseline_sha256':sha(data),
        'reasoning':['CreateRequest checks product active only at create; it allows UnitPrice zero.','TransitionRequest submit currently checks DRAFT/version and updates state; new line/master validation belongs in existing store transaction.','Existing submit handler and UI request carry version and show existing response errors; no new field/route/form requirement.','Extend existing P2P integration scenarios to exercise inactive products, zero price, both failures, valid submit and retained guards. Test filenames are provisional existing-change targets; no new source files invented.','Both results may be collected sequentially within transaction; AND does not mandate goroutines.','No migration or source modification; flow changes excluded from node scope, not labelled negative.']})
    write_json(out/'manifest.json',{'version':'1','repetitions':2,'top_k':10,'label_provenance':'single-source-review','methods':['bm25','semantic','fusion','hybrid'],'scenarios':scenarios})
    print('Frozen P2P checks (six nodes) and unchanged control before inference.')


if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('--repo',type=Path,required=True);p.add_argument('--output',type=Path,required=True);a=p.parse_args();build(a.repo,a.output)
