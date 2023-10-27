<script lang="ts">
  import HThingTable from "$lib/components/HThingTable.svelte";
  import { Heading } from "flowbite-svelte";
</script>


<div class="flex flex-col space-y-2">
  <Heading tag="h3" class="text-center">Discovered Things</Heading>
  <HThingTable></HThingTable>
</div>