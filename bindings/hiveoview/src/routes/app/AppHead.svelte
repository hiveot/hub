<script lang="ts" context="module">
  // import type { IMenuType } from "../../lib/components/HDropDown.svelte";
</script>

<script lang="ts">
  import { Checkbox, DarkMode, Navbar, NavBrand, Toggle, Tooltip } from "flowbite-svelte";
  import DashboardIcon from "$icons/ViewDashboard.svelte";
  import AuthenticatedIcon from "$icons/AccountNetwork.svelte";
  import ConnectedIcon from "$icons/AccountNetworkOff.svelte";
  import DisconnectedIcon from "$icons/CloseNetwork.svelte";
  import RadarIcon from "$icons/Radar.svelte";
  import AboutIcon from "$icons/Information.svelte";
  import ThingsIcon from "$icons/RobotVacuum.svelte";

  import HDropDown from "../../lib/components/HDropDown.svelte";
  import type { IConnectionStatus } from "$lib/hapi/IConnectionStatus.js";
  //

  // Component manages the edit mode
  export let editMode = false;

  function handleAction(menuItem: any): void {
    console.log("action: " + menuItem.label);
  }

  function setEditMode(): void {
    editMode = !editMode;
    console.log("setEditMode to ", editMode);
  }

  // connection status is provided by the layout
  export let connectionStatus: IConnectionStatus;
  $: connIcon = (connectionStatus?.status === "connected") ? ConnectedIcon :
    (connectionStatus?.status === "authenticated") ? AuthenticatedIcon : DisconnectedIcon;

  // the menu must be reactive to show the current edit mode
  $: menuItems = [
    { icon: DashboardIcon, label: "Dashboard", href: "/dashboard" },
    {},
    {
      icon: Checkbox, label: "Edit Mode",
      attr: { checked: editMode },
      onClick: setEditMode
    },
    // { label: "Add Dashboard"},
    {},
    { icon: ThingsIcon, label: "Things", href: "/things" },
    { icon: AboutIcon, label: "About", href: "/about" }
  ];

</script>

<!--Utilize the full width-->
<Navbar navDivClass="mx:auto flex flex-wrap w-full max-w-none justify-between items-center">

  <NavBrand href="/">
    <img src="./logo.svg" alt="logo" class="logo" height="42" />
    <strong class="text-xl uppercase">HiveOT</strong>
  </NavBrand>

  <!-- page tabs-->
  <span class="grow" />
  <!--    Edit button-->
  <Toggle disabled size="small" checked={false}>Edit</Toggle>
  <Tooltip placement="bottom">Toggle dashboard edit mode</Tooltip>

  <!-- One button to rule the night-->
  <DarkMode />
  <Tooltip placement="bottom">Hive at night</Tooltip>

  <!-- network/settings status-->
  <a href="/login">
    <svelte:component this={connIcon}
                      class="h-5 transition duration-1000 delay-150
        {(connectionStatus?.status === 'disconnected') ? 'animate-spin' :''}
        {(connectionStatus?.status === 'authenticated') ? 'dark:text-green-400 text-green-400' :''}"

    />
  </a>
  <Tooltip placement="bottom">{connectionStatus?.statusMessage}</Tooltip>

  <!-- Last, the menu dropdown-->
  <HDropDown
    menu={menuItems}
    onClick={(item)=>{handleAction(item)}}
  />

</Navbar>

<style>
    .logo {
        height: 32px;
    }
</style>
