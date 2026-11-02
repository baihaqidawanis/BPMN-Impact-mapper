async function execute(item: { id: string }, command: string) {
  await api(`/invoices/${item.id}/${command}`, { method: "POST" });
}
