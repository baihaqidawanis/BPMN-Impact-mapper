"""BPMN DI for the active pattern experiments, separate from the business graph."""

import xml.etree.ElementTree as ET

from build_structural_scenarios import q, tag


def set_bounds(bounds, box):
    bounds.attrib.update(dict(zip(("x", "y", "width", "height"), map(str, box))))


def set_label(diagram_element, box):
    label = diagram_element.find(q("bpmndi", "BPMNLabel"))
    if label is None:
        label = ET.SubElement(diagram_element, q("bpmndi", "BPMNLabel"))
    bounds = label.find(q("dc", "Bounds"))
    if bounds is None:
        bounds = ET.SubElement(label, q("dc", "Bounds"))
    set_bounds(bounds, box)


class Layout:
    def __init__(self, root, prefix, right, lane_bottoms):
        self.prefix = prefix
        self.flows = {flow.get("id"): flow for flow in root.iter(tag("sequenceFlow"))}
        self.shapes = {shape.get("bpmnElement"): shape for shape in root.iter(q("bpmndi", "BPMNShape"))}
        self.edges = {edge.get("bpmnElement"): edge for edge in root.iter(q("bpmndi", "BPMNEdge"))}
        self.boxes = {}
        participant = root.find(".//" + tag("participant"))
        set_bounds(self.shapes[participant.get("id")].find(q("dc", "Bounds")),
                   (20, 40, right - 20, lane_bottoms[-1] - 40))
        top = 40
        for lane, bottom in zip(root.iter(tag("lane")), lane_bottoms, strict=True):
            set_bounds(self.shapes[lane.get("id")].find(q("dc", "Bounds")), (50, top, right - 50, bottom - top))
            top = bottom

    def identifier(self, suffix):
        return suffix if suffix in self.shapes or suffix in self.edges else self.prefix + suffix

    def node(self, suffix, x, center, width=None, caption=None):
        identifier = self.identifier(suffix)
        shape = self.shapes[identifier]
        bounds = shape.find(q("dc", "Bounds"))
        width = width or float(bounds.get("width"))
        height = float(bounds.get("height"))
        self.boxes[identifier] = (x, center - height / 2, width, height)
        set_bounds(bounds, self.boxes[identifier])
        if caption:
            set_label(shape, caption)
        elif shape.find(q("bpmndi", "BPMNLabel")) is not None:
            set_label(shape, (x + width / 2 - 65, center + height / 2 + 7, 130, 31))

    def port(self, identifier, side):
        x, y, width, height = self.boxes[identifier]
        return {
            "left": (x, y + height / 2),
            "right": (x + width, y + height / 2),
            "top": (x + width / 2, y),
            "bottom": (x + width / 2, y + height),
            # Lower-left quadrant of a gateway diamond.
            "diamond-lower-left": (x + width / 4, y + height * 3 / 4),
        }[side]

    def edge(self, suffix, via=(), source="right", target="left", caption=None):
        identifier = self.identifier(suffix)
        flow = self.flows[identifier]
        edge = self.edges[identifier]
        points = (self.port(flow.get("sourceRef"), source), *via, self.port(flow.get("targetRef"), target))
        if any((ax, ay) == (bx, by) or (ax != bx and ay != by)
               for (ax, ay), (bx, by) in zip(points, points[1:])):
            raise ValueError(f"Invalid orthogonal waypoints for {identifier}: {points}")
        for point in edge.findall(q("di", "waypoint")):
            edge.remove(point)
        for index, (x, y) in enumerate(points):
            edge.insert(index, ET.Element(q("di", "waypoint"), x=str(x), y=str(y)))
        if flow.get("name"):
            if caption is None:
                raise ValueError(f"Missing label placement for {identifier}")
            set_label(edge, caption)


def skip_layout(root, prefix):
    layout = Layout(root, prefix, right=2750, lane_bottoms=(195, 612))
    layout.node("start", 90, 120)
    layout.node("input", 220, 120, 205)
    layout.node("check", 500, 373, 208)
    layout.node("valid", 780, 373, caption=(743, 422, 130, 29))
    layout.node("CheckZeroDelta", 970, 373, 192)
    layout.node("ZeroDeltaDecision", 1225, 373, caption=(1100, 423, 140, 29))
    layout.node("effect0", 1475, 373, 186)
    layout.node("effect1", 1725, 373, 188)
    layout.node("Flow_6_prepare_result", 970, 253, 198)
    layout.node("CloseWithoutAdjustment", 1500, 537, 205)
    layout.node("result_merge", 2090, 373)
    layout.node("result_activity", 2315, 373, 198)
    layout.node("done", 2630, 373)

    layout.edge("Flow_0")
    layout.edge("Flow_1", ((460, 120), (460, 373)))
    layout.edge("Flow_2")
    layout.edge("Flow_3", caption=(857, 335, 80, 29))
    layout.edge("Flow_4")
    layout.edge("Flow_5")
    layout.edge("Flow_6", ((807.5, 253),), source="top", caption=(825, 210, 137, 35))
    layout.edge("result_to_end")
    layout.edge("DeltaChecked")
    layout.edge("SubmitForApproval", caption=(1291, 324, 176, 38))
    layout.edge("BypassApproval", ((1252.5, 537),), source="bottom", caption=(1285, 494, 150, 34))
    layout.edge("BypassCompleted", ((2117.5, 537),), target="bottom")
    layout.edge("Flow_6_prepared", ((2117.5, 253),), target="top")
    layout.edge("result_joined")


def redo_layout(root, prefix):
    layout = Layout(root, prefix, right=2500, lane_bottoms=(210, 790, 1000))
    layout.node("start", 90, 125, caption=(58, 152, 88, 31))
    layout.node("LoadFirstReview", 190, 395, 184)
    layout.node("ReviewEntryJoin", 435, 395, caption=(291, 444, 150, 28))
    layout.node("input", 540, 125, 240)
    layout.node("check", 830, 395, 250)
    layout.node("valid", 1145, 395, caption=(1187, 444, 140, 28))
    layout.node("positive", 1300, 395)
    layout.node("negative", 1300, 528)
    layout.node("ReturnPRForRevision", 1300, 665, 225)
    layout.node("EditReturnedPR", 1600, 889, 220)
    layout.node("ResubmitRevisedPR", 1860, 662, 225)
    layout.node("RedoValidationFailure", 1450, 259, 220)
    layout.node("DecisionAuditJoin", 1535, 395, caption=(1480, 436, 74, 34))
    layout.node("effect1", 1640, 395, 205)
    layout.node("ReviewOutcomeJoin", 1950, 395)
    layout.node("result_activity", 2080, 395, 210)
    layout.node("done", 2380, 395)

    layout.edge("Flow_0", ((150, 125), (150, 395)))
    layout.edge("FirstReviewReady")
    layout.edge("EnterReview", ((515, 395), (515, 125)))
    layout.edge("Flow_1", ((805, 125), (805, 395)))
    layout.edge("Flow_2")
    layout.edge("Flow_4", ((1172.5, 259),), source="top",
                caption=(1215, 221, 220, 32))
    layout.edge("ReviewValidationFailureReady", ((1977.5, 259),), target="top")
    layout.edge("Flow_5", caption=(1205, 357, 88, 30))
    layout.edge("Flow_6", ((1172.5, 528),), source="bottom", caption=(1203, 491, 90, 30))
    layout.edge("RevisionRequested", ((1158.75, 665),), source="diamond-lower-left",
                caption=(1170, 627, 118, 30))
    layout.edge("Flow_7")
    layout.edge("Flow_8", ((1562.5, 528),), target="bottom")
    layout.edge("DecisionReadyForAudit")
    layout.edge("Flow_3", caption=(1826, 308, 145, 38))
    layout.edge("ReturnReviewOutcome")
    layout.edge("result_to_end")
    layout.edge("ReturnedToStaff", ((1565, 665), (1565, 889)))
    layout.edge("ApplyRevision", ((1840, 889), (1840, 662)))
    layout.edge("BackToReview", ((1972.5, 963), (462.5, 963)), source="bottom", target="bottom",
                caption=(970, 925, 250, 32))


def switch_layout(root, prefix):
    layout = Layout(root, prefix, right=3040, lane_bottoms=(200, 671))
    layout.node("start", 90, 120)
    layout.node("input", 220, 120, 226)
    layout.node("check", 515, 360, 224)
    layout.node("valid", 815, 360, caption=(860, 407, 130, 29))
    layout.node("CheckProductsAtApproval", 955, 360, 191)
    layout.node("ApprovalProductDecision", 1215, 360, caption=(1260, 409, 155, 29))
    layout.node("positive", 1445, 360, 180)
    layout.node("PrepareInactiveReason", 1435, 510, 185)
    layout.node("KeepManualRejectionReason", 1390, 611, 180)
    layout.node("RejectionPathJoin", 1680, 510, caption=(1628, 548, 73, 33))
    layout.node("negative", 1785, 510, 180)
    layout.node("DecisionAuditJoin", 2040, 360, caption=(1980, 402, 79, 36))
    layout.node("SwitchValidationFailure", 1480, 254, 225)
    layout.node("effect1", 2155, 360, 200)
    layout.node("ReviewOutcomeJoin", 2490, 360)
    layout.node("result_activity", 2625, 360, 200)
    layout.node("done", 2920, 360)

    layout.edge("Flow_0")
    layout.edge("Flow_1", ((480, 120), (480, 360)))
    layout.edge("Flow_2")
    layout.edge("Flow_4", ((842.5, 254),), source="top",
                caption=(884, 215, 235, 32))
    layout.edge("ReviewValidationFailureReady", ((2517.5, 254),), target="top")
    layout.edge("Flow_5", caption=(874, 327, 78, 27))
    layout.edge("Flow_6", ((842.5, 611),), source="bottom", caption=(1030, 620, 130, 27))
    layout.edge("ProductsChecked")
    layout.edge("ContinueApproval", caption=(1275, 319, 160, 35))
    layout.edge("SwitchToRejection", ((1242.5, 510),), source="bottom", caption=(1276, 467, 147, 37))
    layout.edge("ApplyExistingRejection")
    layout.edge("ManualRejectionReady", ((1707.5, 611),), target="bottom")
    layout.edge("ApplyRejectionOutcome")
    layout.edge("Flow_7")
    layout.edge("Flow_8", ((2067.5, 510),), target="bottom")
    layout.edge("DecisionReadyForAudit")
    layout.edge("Flow_3", caption=(2360, 315, 122, 37))
    layout.edge("ReturnReviewOutcome")
    layout.edge("result_to_end")


def nfc_layout(root, prefix):
    layout = Layout(root, prefix, right=3810, lane_bottoms=(195, 687, 946))
    layout.node("start", 90, 119)
    layout.node("input", 210, 119, 194)
    layout.node("check", 475, 390, 180)
    layout.node("valid", 725, 390, caption=(689, 425, 130, 29))
    layout.node("LockApprovalRoute", 855, 390, 188)
    layout.node("effect0", 1105, 390, 186)
    layout.node("effect1", 1355, 390, 180)
    layout.node("ReturnSubmitReceipt", 1605, 390, 190)
    layout.node("RequestLockedReview", 1860, 815, 200)
    layout.node("LoadLockedApprovalRoute", 2125, 390, 190)
    layout.node("LockedRouteDecision", 2385, 390, caption=(2434, 433, 150, 40))
    layout.node("OpenStandardReview", 2590, 390, 200)
    layout.node("OpenEscalatedReview", 2590, 550, 200)
    layout.node("RejectUnknownRoute", 2590, 302, 200)
    layout.node("NFCSubmitValidationFailure", 1090, 250, 220)
    layout.node("LockedReviewJoin", 2850, 390, caption=(2790, 431, 78, 32))
    layout.node("PrepareLockedReviewOutcome", 2960, 390, 190)
    layout.node("NFCOutcomeJoin", 3240, 390)
    layout.node("result_activity", 3390, 390, 200)
    layout.node("done", 3690, 390)

    layout.edge("Flow_0")
    layout.edge("Flow_1", ((435, 119), (435, 390)))
    layout.edge("Flow_2")
    layout.edge("Flow_3", caption=(785, 355, 62, 29))
    layout.edge("Flow_4")
    layout.edge("Flow_5")
    layout.edge("Flow_6", ((752.5, 250),), source="top",
                caption=(804, 258, 252, 34))
    layout.edge("NFCSubmitFailureReady", ((3267.5, 250),), target="top")
    layout.edge("RouteLocked")
    layout.edge("ContinueToReview", ((1826, 390), (1826, 815)), caption=(1624, 625, 180, 35))
    layout.edge("ReadLockedRoute", ((2092, 815), (2092, 390)))
    layout.edge("RouteLoaded")
    layout.edge("StoredStandard", caption=(2445, 351, 139, 30))
    layout.edge("StoredEscalated", ((2412.5, 550),), source="bottom",
                caption=(2435, 508, 145, 34))
    layout.edge("MissingRoute", ((2412.5, 302),), source="top", caption=(2428, 263, 145, 34))
    layout.edge("OpenStandardReviewDone")
    layout.edge("OpenEscalatedReviewDone", ((2877.5, 550),), target="bottom")
    layout.edge("RejectUnknownRouteDone", ((2877.5, 302),), target="top")
    layout.edge("PrepareLockedOutcome")
    layout.edge("LockedOutcomeReady")
    layout.edge("ReturnNFCOutcome")
    layout.edge("result_to_end")
