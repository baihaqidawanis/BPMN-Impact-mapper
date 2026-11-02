"""Explicit case-study review decisions, never production retrieval rules.

Source files/functions were selected by static inspection. Needles anchor those
decisions and fail on drift; they do not discover semantic relationships.
"""
from copy import deepcopy
from build_baseline_traceability import SPECS as STAFF_P2P


SPECS = deepcopy(STAFF_P2P)


def add(key, module, store, handler, ui, component, inputs, checks, effects, **extra):
    SPECS[key] = dict(module=module, store=store, handler=handler, ui=ui,
                      component=component, input=inputs, check=checks,
                      effects=effects, **extra)


add('pr-review', 'purchasing', 'TransitionRequest', 'prAction',
    'purchase-request-panel', 'PurchaseRequestPanel',
    ['action(document, "approve")', 'action(document, "reject")'],
    ['from != "SUBMITTED"', 'authorization.CanSelfReview('],
    [[], ['reviewer_id=CASE', '"PUR-PR-003"', 'audit(ctx', 'tx.Commit(']],
    branches={'positive': ['d.Status = "APPROVED"'], 'negative': ['d.Status = "REJECTED"', 'reason == ""']})
add('po-issue', 'purchasing', 'IssueOrder', 'issue', 'purchase-order-panel', 'PurchaseOrderPanel',
    ['/issue`'], ["status='DRAFT'", "role='SUPER_ADMIN' AND status='ACTIVE'", 'version=$2'],
    [["status='ISSUED'"], ['"PUR-PO-002"', 'tx.Commit(']])
add('po-cancel', 'purchasing', 'CancelOrder', 'cancelOrder', 'purchase-order-panel', 'PurchaseOrderPanel',
    ['/cancel`', 'disabled={!reason}'], ['reason == ""', 'previous != "DRAFT"', "g.status='POSTED'"],
    [["status='CANCELLED'", "status='APPROVED'"], ['"CANCEL_PO"', '"REOPEN_PR_AFTER_PO_CANCEL"', 'tx.Commit(']])
add('gr-post', 'purchasing', 'PostReceipt', 'postReceipt', 'goods-receipt-panel', 'GoodsReceiptPanel',
    ['/post`'], ['d.Status != "DRAFT"', 'd.Version != version', 'authorization.CanSelfReview(', 'l.qty > l.remaining'],
    [['INSERT INTO stock_balances', 'INSERT INTO stock_movements'],
     ["status='POSTED'", 'UPDATE purchase_orders SET status=$1'], ['"PUR-GR-002"', 'tx.Commit(']])
add('gr-reverse', 'purchasing', 'ReverseReceipt', 'reverseReceipt', 'goods-receipt-panel', 'GoodsReceiptPanel',
    ['/reverse`', 'disabled={!reason}'], ['reason == ""', 'd.Status != "POSTED"', "status<>'VOID'", 'balance < l.qty'],
    [['quantity=quantity-$1', "'GOODS_RECEIPT_REVERSAL'", 'received_quantity=received_quantity-$1'],
     ["status='REVERSED'", "THEN'PARTIALLY_RECEIVED'ELSE'ISSUED'"], ['"ACT-PUR-GR-REVERSE"', 'tx.Commit(']])
add('ap-match', 'finance', 'MatchSupplierInvoice', 'matchSupplierInvoice', 'finance-workspace', 'FinancePanel',
    ['`${invoicePrefix}/${item.id}/match`'], ['"MATCH_PENDING"', 'authorization.CanSelfReview(', 'domain.ExactMatch('],
    [[], ['mismatch_reason=NULLIF', '"PUR-INV-002"', 'tx.Commit(']],
    branches={'positive': ['to := "POSTED"'], 'negative': ['to = "MATCH_EXCEPTION"', 'domain.ExactMatch(']})
add('ap-invoice-void', 'finance', 'VoidSupplierInvoice', 'voidSupplierInvoice', 'finance-workspace', 'FinancePanel',
    ['`${invoicePrefix}/${item.id}/void`', 'disabled={!voidReason}'], ['reason == ""', "i.status='POSTED'", "p.status='POSTED'"],
    [["status='VOID'", 'voided_by=$3'], ['"ACT-FIN-SINV-VOID"', 'tx.Commit(']])
add('ap-payment-post', 'finance', 'PostSupplierPayment', 'postSupplierPayment', 'finance-workspace', 'FinancePanel',
    ['`${paymentPrefix}/${item.id}/post`'], ['"DRAFT"', 'document.Version != version', 'authorization.CanSelfReview(', 'domain.ValidatePayment(', 'item.amount > item.outstanding'],
    [['UPDATE supplier_invoices SET outstanding=$2', 'status = "PAID"'],
     ["status='CLOSED'", "status='COMPLETED'"], ["UPDATE supplier_payments SET status='POSTED'", '"PUR-PAY-002"', 'tx.Commit(']])
add('ap-payment-void', 'finance', 'VoidSupplierPayment', 'voidSupplierPayment', 'finance-workspace', 'FinancePanel',
    ['`${paymentPrefix}/${item.id}/void`', 'disabled={!voidReason}'], ['reason == ""', 'document.Status != "POSTED"'],
    [['UPDATE supplier_invoices SET outstanding=$2', 'status = "POSTED"'],
     ["status='RECEIVED'", "status='CONVERTED_TO_PO'"], ["UPDATE supplier_payments SET status='VOID'", '"ACT-FIN-SPAY-VOID"', 'tx.Commit(']])

add('adj-create', 'inventory', 'CreateAdjustment', 'create', 'inventory-adjustment-panel', 'InventoryAdjustmentPanel',
    ['onSubmit={create}', 'countedQuantity: Number(countedQuantity)'],
    ['len(lines) == 0', 'FROM warehouses', 'l.CountedQuantity < 0', 'FROM products'],
    [['INSERT INTO inventory_adjustments', 'snapshot_quantity,counted_quantity'], ['"INV-ADJ-001"', 'tx.Commit(']])
add('adj-submit', 'inventory', 'TransitionAdjustment', 'action', 'inventory-adjustment-panel', 'InventoryAdjustmentPanel',
    ['void action("submit")'], ['d.Version != version', 'case "submit":', 'from != "DRAFT"'],
    [['d.Status = "SUBMITTED"', 'UPDATE inventory_adjustments'], ['"INV-ADJ-002"', 'audit(ctx', 'tx.Commit(']], service='Transition')
add('adj-review', 'inventory', 'TransitionAdjustment', 'action', 'inventory-adjustment-panel', 'InventoryAdjustmentPanel',
    ['action("approve")', 'action("reject")'], ['from != "SUBMITTED"', 'authorization.CanSelfReview(', '!snapshotCurrent'],
    [[], ['reviewer_id=CASE', '"INV-ADJ-003"', 'audit(ctx', 'tx.Commit(']], service='Transition',
    branches={'positive': ['d.Status = "APPROVED"'], 'negative': ['d.Status = "REJECTED"', 'reason == ""']})
add('adj-cancel', 'inventory', 'TransitionAdjustment', 'action', 'inventory-adjustment-panel', 'InventoryAdjustmentPanel',
    ['action("cancel")', 'disabled={!reason'], ['from != "DRAFT" && from != "APPROVED"', 'actor != d.CreatorID && !admin', 'reason == ""'],
    [['d.Status = "CANCELLED"', 'UPDATE inventory_adjustments'], ['"ACT-INV-ADJ-CANCEL"', 'audit(ctx', 'tx.Commit(']], service='Transition')
add('adj-post', 'inventory', 'PostAdjustment', 'action', 'inventory-adjustment-panel', 'InventoryAdjustmentPanel',
    ['action("post")'], ['d.Status != "APPROVED"', 'd.Version != version', 'current != l.snapshot'],
    [['delta := l.counted - l.snapshot', 'delta == 0', 'INSERT INTO stock_balances', 'INSERT INTO stock_movements'],
     ["status='COMPLETED'"], ['`INV-ADJ-004`', '`INV-ADJ-005`', 'tx.Commit(']], service='Post')
add('adj-reverse', 'inventory', 'ReverseAdjustment', 'action', 'inventory-adjustment-panel', 'InventoryAdjustmentPanel',
    ['action("reverse")', 'disabled={!reason}'], ['reason == ""', 'd.Status != "COMPLETED"', 'balance-l.qty < 0'],
    [['quantity=quantity-$1', "'STOCK_ADJUSTMENT_REVERSAL'"], ["status='REVERSED'", '"ACT-INV-ADJ-REVERSE"', 'tx.Commit(']], service='Reverse')
for key, func, handler, route, table in (
    ('inv-balances', 'Balances', 'balances', '/inventory/balances', 'stock_balances'),
    ('inv-movements', 'Movements', 'movements', '/inventory/movements', 'stock_movements'),
    ('inv-adjustments', 'ListAdjustments', 'listAdjustments', '/inventory-adjustments', 'inventory_adjustments'),
):
    panel = key == 'inv-adjustments'
    add(key, 'inventory', func, handler,
        'inventory-adjustment-panel' if panel else 'inventory-workspace',
        'InventoryAdjustmentPanel' if panel else 'InventoryWorkspace',
        ['"'+route+'"'], [], [], read=['FROM '+table])
for key, func, handler, checks, effects, route in (
    ('delivery-post', 'PostDelivery', 'post', ['d.Status != "DRAFT"', 'd.Version != version', 'l.qty > l.remaining', 'balance < l.qty'],
     [['quantity=quantity-$1', "'SALES_DELIVERY'"], ["status='POSTED'", 'UPDATE sales_orders SET status=$1'], ['"SAL-DLV-002"', 'tx.Commit(']], '/post`'),
    ('delivery-reverse', 'ReverseDelivery', 'reverse', ['reason == ""', 'd.Status != "POSTED"', "status<>'VOID'"],
     [['quantity=quantity+$1', "'SALES_DELIVERY_REVERSAL'", 'delivered_quantity=delivered_quantity-$1'],
      ["status='REVERSED'", "THEN 'PARTIALLY_DELIVERED' ELSE 'APPROVED'"], ['"ACT-SAL-DLV-REVERSE"', 'tx.Commit(']], '/reverse`'),
):
    add(key, 'sales', func, handler, 'delivery-panel', 'DeliveryPanel', [route], checks, effects)

add('master-list', 'masterdata', 'List', 'list', 'master-data-panel', 'MasterDataPanel',
    ['`/master-data/${resource}`'], [], [], read=['table(k)', 'SELECT id,', 'ORDER BY'])
add('master-create', 'masterdata', 'Create', 'create', 'master-data-panel', 'MasterDataPanel',
    ['onSubmit={create}', 'JSON.stringify({ code, name })'],
    ['table(r.Kind)', "code='PCS' AND active", 'tag.RowsAffected() != 1'],
    [['INSERT INTO products', '"INSERT INTO "+t'], ["'CREATE_MASTER_DATA'", 'tx.Commit(']])
add('master-status', 'masterdata', 'SetActive', 'status', 'master-data-panel', 'MasterDataPanel',
    ['toggle(item)', 'version: item.version'], ['table(k)', 'tag.RowsAffected() != 1'],
    [['SET active=$1,version=version+1', 'WHERE id=$2 AND version=$3'], ["'SET_MASTER_DATA_STATUS'", 'tx.Commit(']])
add('settings-get', 'masterdata', 'Get', 'ServeHTTP', 'settings-form', 'SettingsForm',
    ['api<Settings>("/settings/organization")'], [], [], read=['SELECT organization_name', 'WHERE singleton'], settings=True)
add('settings-update', 'masterdata', 'Update', 'ServeHTTP', 'settings-form', 'SettingsForm',
    ['method:"PATCH"', 'onSubmit={save}'], ['tag.RowsAffected() != 1'],
    [['UPDATE organization_settings SET', 'WHERE singleton'], ["'UPDATE_SETTINGS'", 'tx.Commit(']], settings=True)

# Read-only review links to existing test source, not a claim tests were run or
# every role/branch has been tested.
TESTS = {}
for key in ('pr-create', 'pr-submit', 'pr-review', 'po-convert', 'po-issue', 'gr-create', 'gr-post', 'ap-invoice', 'ap-payment'):
    TESTS[key] = ('operations_test.go', 'TestP2PPartialReceiptPostsImmutableSourceLinkedLedger', SPECS[key]['store']+'(')
TESTS['pr-copy'] = ('operations_test.go', 'TestCopyRejectedDocumentsRequiresActiveMasters', 'purchasing.CopyRequest(')
for key in ('ap-resolve', 'ap-match', 'ap-payment-post', 'ap-payment-void', 'ap-invoice-void'):
    TESTS[key] = ('finance_test.go', 'TestP2PConsolidatedInvoiceExactMatchPartialPaymentCloseAndVoid', SPECS[key]['store']+'(')
for key in ('adj-create', 'adj-submit', 'adj-review', 'adj-post', 'adj-cancel'):
    TESTS[key] = ('operations_test.go', 'TestApprovedInventoryAdjustmentCanRecoverFromStaleSnapshot', 's.'+SPECS[key].get('service', SPECS[key]['store'])+'(')
