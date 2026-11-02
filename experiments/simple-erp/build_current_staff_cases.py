"""Freeze normalized Staff pairs and provisional references before inference."""
import argparse
import copy
import json
from pathlib import Path
import xml.etree.ElementTree as ET
from build_structural_scenarios import tag, node, flow, diagram, sha


def build(repo, output, case_prefix='normalized-staff'):
    output.mkdir(parents=True, exist_ok=False)
    data=(repo/'docs/bpm-haqi/roles/staff/inventory/level-3/role-staff-l3-adj-submit.bpmn').read_bytes()
    base=ET.fromstring(data);prefix='role_staff_l3_adj_submit_'
    # Outcome handling is an existing activity, not an assumed merge gateway.
    # Canonical normalization may remove that redundant gateway.
    outcome=prefix+'result_activity'
    if not any(e.get('id')==outcome for e in base.iter()):
        raise ValueError('Canonical baseline has no reviewed result activity')
    store='backend/internal/inventory/adapter/postgres/store.go'
    panel='frontend/features/inventory/inventory-adjustment-panel.tsx'
    test='backend/test/integration/operations_test.go'
    references={
        'add-task':('ADD_TASK','ReasonReview',[panel,store,test]),
        'add-xor':('ADD_GATEWAY','ReasonDecision',[panel,store,test]),
        'add-and':('ADD_GATEWAY','MasterChecksSplit',[store,test]),
        'combined':('ADD_GATEWAY','ReasonDecision',[panel,store,test]),
    }
    scenarios=[]
    for name in ['add-task','add-xor','add-and','combined','unchanged-control']:
        folder=output/name;folder.mkdir()
        (folder/'before.bpmn').write_bytes(data)
        root=copy.deepcopy(base);p=root.find(tag('process'));staff,system=p.findall('.//'+tag('lane'))[:2]
        bridge=next(f for f in p.findall(tag('sequenceFlow')) if f.get('sourceRef')==prefix+'input' and f.get('targetRef')==prefix+'check')
        entry=prefix+'check'
        if name in ['add-and','combined']:
            node(p,'parallelGateway','MasterChecksSplit','Run both stock adjustment master-data checks',system,gatewayDirection='Diverging')
            node(p,'parallelGateway','MasterChecksJoin','Both validation results collected',system,gatewayDirection='Converging')
            for id,label in [('CheckWarehouse','Validate warehouse is still active'),('CheckProducts','Validate all products are still active')]:
                node(p,'serviceTask',id,label,system);flow(p,id+'Start','MasterChecksSplit',id);flow(p,id+'Done',id,'MasterChecksJoin')
            node(p,'serviceTask','CombineValidation','Aggregate validation results without changing stock or status',system)
            flow(p,'CombineResults','MasterChecksJoin','CombineValidation')
            node(p,'exclusiveGateway','MasterDecision','Both checks passed?',system,gatewayDirection='Diverging',default='InvalidMasters')
            flow(p,'DecideResults','CombineValidation','MasterDecision')
            flow(p,'ValidMasters','MasterDecision',prefix+'check','warehouse active and all products active')
            flow(p,'InvalidMasters','MasterDecision',outcome)
            entry='MasterChecksSplit'
        if name in ['add-xor','combined']:
            if name=='combined':
                # Read the IDs required by both checks before the AND split;
                # avoid a direct XOR->AND connection in the local convention.
                node(p,'serviceTask','CollectMasterContext','Read warehouse and product references for both submit checks',system)
                flow(p,'MasterContextReady','CollectMasterContext',entry)
                entry='CollectMasterContext'
            node(p,'exclusiveGateway','ReasonDecision','Is stock adjustment submit reason provided?',system,gatewayDirection='Diverging',default='ReasonMissing')
            flow(p,'ReasonProvided','ReasonDecision',entry,'reason is not empty')
            flow(p,'ReasonMissing','ReasonDecision',outcome)
            entry='ReasonDecision'
        if name=='add-task':
            node(p,'userTask','ReasonReview','Enter and validate stock adjustment reason before submit',staff)
            flow(p,'ReasonChecked','ReasonReview',entry);entry='ReasonReview'
        bridge.set('targetRef',entry)
        if name=='unchanged-control':after=data
        else:
            diagram(root);ET.indent(root);after=ET.tostring(root,encoding='utf-8',xml_declaration=True)
        (folder/'after.bpmn').write_bytes(after)
        cases=[]
        if name in references:
            kind,id,files=references[name]
            cases=[{'change_type':kind,'element_id':id,'expected':[{'file':f,'required_action':'MODIFY'} for f in files]}]
        truth=json.dumps({'version':'1','cases':cases},indent=2)+'\n'
        (folder/'reference.json').write_bytes(truth.encode('utf-8'))
        requirement=('No BPMN or software change.' if name=='unchanged-control' else
            'Require non-empty reason before adjustment submit.' if name in ['add-task','add-xor'] else
            'At submit, revalidate active warehouse AND all active products; collect both results, reject on failure, retain state/access/version guards.' if name=='add-and' else
            'Require submit reason and both active-master validations. Any failure must not submit; existing state/access/version guards remain.')
        evidence={'requirement':requirement,'scope':'one selected primary change; other changes not labelled or counted as negatives',
            'reference_status':'provisional source-derived diagnostic; prepared before inference, not independent ground truth',
            'source_hashes':{f:sha((repo/f).read_bytes()) for f in [store,panel,test]},
            'reasoning':'Submit currently only checks DRAFT/version. Reason field exists but UI submits empty reason. Master active checks during create do not ensure they remain active at submit. Operations integration tests exercise existing submit/guards. No schema or new endpoint required.'}
        (folder/'requirements.md').write_text('# '+name+'\n\n'+requirement+'\n\n'+evidence['reasoning']+'\n\nReference scope: '+evidence['scope']+'.\nReference status: '+evidence['reference_status']+'.\n',encoding='utf-8')
        (folder/'source-review.json').write_text(json.dumps(evidence,indent=2)+'\n')
        scenarios.append({'id':case_prefix+'-'+name,'repository_path':str(repo),'repository_name':'Simple-ERP','revision':'local-'+case_prefix,
            'old_bpmn_path':(folder/'before.bpmn').as_posix(),'old_bpmn_sha256':sha(data),
            'new_bpmn_path':(folder/'after.bpmn').as_posix(),'new_bpmn_sha256':sha(after),
            'ground_truth_path':(folder/'reference.json').as_posix(),'ground_truth_sha256':sha(truth.encode()),
            'truth_scope':'complete' if name=='unchanged-control' else 'subset'})
    manifest={'version':'1','repetitions':2,'top_k':10,'label_provenance':'single-source-review',
              'methods':['bm25','semantic','fusion','hybrid'],'scenarios':scenarios}
    # Relative local repository location; no absolute path in shareable manifest.
    for s in scenarios:s['repository_path']='../Simple-ERP'
    (output/'manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')


if __name__=='__main__':
    parser=argparse.ArgumentParser();parser.add_argument('--repo',type=Path,required=True);parser.add_argument('--output',type=Path,required=True)
    parser.add_argument('--case-prefix',default='normalized-staff')
    a=parser.parse_args();build(a.repo,a.output,a.case_prefix)
