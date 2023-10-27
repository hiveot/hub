<script lang="ts">
  import "../app.postcss";

  import AppHead from "./app/AppHead.svelte";
  import { onMount } from "svelte";
  import type { PageData } from "./$types.js";
  // import { hapi } from "$lib/hapi/HubAPI.server.js";

  // data exported by layout.server.ts load
  /** @type {import("./$types").PageData} */
  export let data;
  $: loginID = data.loginID;


  // console.info("onMount. data=", data);
  onMount(async () => {
    console.info("onMount. data=", data);
  });

</script>


<div class="flex flex-col max-auto w-full h-full">
  <!--  <AppHead connectionStatus="{data.conStat}"></AppHead>-->
  <AppHead connectionStatus="{data?.conStat}"></AppHead>

  <!-- Router Slot -->
  <slot />
  <p>connected:{data?.conStat?.statusMessage}</p>

  <!--	<svelte:fragment slot="footer">Footer</svelte:fragment>-->
</div>
