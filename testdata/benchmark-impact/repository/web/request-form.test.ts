import { submitRequest } from "./request-form";

export function testSubmitRequest() {
  return submitRequest(100).status === "SUBMITTED";
}
