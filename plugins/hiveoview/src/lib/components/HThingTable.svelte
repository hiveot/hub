<script>
  // table with discovered things
  import { Table, TableBody, TableBodyRow, TableBodyCell, TableHead, TableHeadCell } from "flowbite-svelte";
</script>

<Table>
  <TableHead>
    <TableHeadCell>Publisher ID</TableHeadCell>
    <TableHeadCell>Thing ID</TableHeadCell>
    <TableHeadCell>Title</TableHeadCell>
  </TableHead>

  <TableBody>
    <TableBodyRow>
      <TableBodyCell>Pub1</TableBodyCell>
      <TableBodyCell>Thing1</TableBodyCell>
      <TableBodyCell>Thing one</TableBodyCell>
    </TableBodyRow>
  </TableBody>
</Table>