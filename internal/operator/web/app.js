const ND_TOKEN = (function(){
  // Prefer injected literal from /static/app.js; fallback meta on index.html.
  const injected = '/*__ND_TOKEN__*/';
  if (injected && injected.indexOf('__ND_TOKEN__') < 0) return injected;
  const m = document.querySelector('meta[name="nd-token"]');
  const fromMeta = m ? (m.getAttribute('content') || '') : '';
  if (fromMeta && fromMeta.indexOf('__ND_TOKEN__') < 0) return fromMeta;
  return '';
})();
function settings(){
  try{
    const o=JSON.parse(localStorage.getItem('nd_op_settings')||'{}');
    // session never from localStorage (XSS residual)
    try{ const s=sessionStorage.getItem('nd_op_session'); if(s) o.node_session=s; else delete o.node_session; }catch(e){ delete o.node_session; }
    return o;
  }catch(e){return{}}
}
function saveSettingsObj(o){
  const copy=Object.assign({}, o);
  const tok=copy.node_session||'';
  delete copy.node_session;
  localStorage.setItem('nd_op_settings', JSON.stringify(copy));
  try{ if(tok) sessionStorage.setItem('nd_op_session', tok); else sessionStorage.removeItem('nd_op_session'); }catch(e){}
}
function apiBase(){ return (settings().api_base||'http://127.0.0.1:8787').replace(/\/$/,''); }

const I18N = {
en:{
  app_title:'netductor-op', nav_installer:'Deploy', nav_control:'Operate', nav_settings:'Settings', meta_line:'Mac client · node API', nav_grp_fleet:'Fleet', nav_servers:'Servers', nav_sites:'Sites', servers_note:'Nodes of the control plane. Click a card for details and actions.', sites_note:'Edge sites — topology, zones, metrics. Add site opens the OpenWrt wizard.', sites_add:'+ Add site', sites_empty:'No enrolled sites yet. Add a site to run the OpenWrt wizard.', sites_offline:'Need tunnel + session to load sites from primary.', card_openwrt:'OpenWrt / site', card_openwrt_d:'Probe → modules → provision', card_mt_d:'RSC push / site shell', actions:'Actions', nav_grp_deploy:'Deploy', nav_grp_operate:'Operate', nav_collapse:'Collapse', l_port:'Port', b_gh_token:'GitHub token',
  tab_installer:'Installer', tab_control:'Control', tab_settings:'Settings', tab_updates:'Updates',
  upd_local:'Local', upd_latest:'Latest', upd_pick:'Release', upd_apply:'Apply on primary', upd_refresh:'Refresh', upd_available:'update available', upd_ok:'up to date',
  sub_fleet:'Fleet', sub_primary:'Primary', sub_build:'Build-host', sub_secondary:'Secondary', sub_edge:'OpenWrt', sub_site:'MikroTik', sub_creds:'Credentials',
  inst_note:'Deploy from this Mac over SSH.',
  l_phost:'Primary host', l_ppass:'Primary password', l_shost:'Secondary host', l_spass:'Secondary password',
  l_domain:'Domain label (optional org tag)', l_domain_primary:'Primary domain', l_domain_vpn:'Secondary domain', l_domain_redirect:'Redirect URL', l_ssh_port:'SSH port', l_redir_port:'Redirect HTTPS', l_mtls_port:'Agent mTLS', l_lampac_port:'Lampac', l_le:'LE email', l_sni:'SNI (Reality camouflage)', l_key:'SSH key', l_host:'Host', l_pass:'Password',
  l_pkey:'Primary key', l_user:'User', l_arch:'Agent arch', l_server_url:'Server URL (mTLS)', l_keypass:'Key passphrase',
  l_lan:'LAN IP (optional)', l_wan:'WAN proto', l_ssid:'Wi-Fi SSID', l_wifikey:'Wi-Fi key',
  l_site_id:'Site id', l_name:'Name', l_rpi:'RPi id', l_mtid:'MikroTik id', l_rpilan:'RPi LAN gateway',
  l_mthost:'MT host', l_mtuser:'MT user', l_mtpass:'MT password', l_mtport:'MT port', l_opkey:'Operator key',
  l_role:'Role', l_tg_token:'TG bot token', l_tg_admin:'TG admin user id',
  c_do_primary:'Primary', c_do_secondary:'Secondary', c_do_build:'Build-host', c_lampac:'Lampac', c_git:'Git', c_tg:'Telegram bot', 
  c_netcfg:'Configure network', c_guest:'Guest Wi-Fi', c_reboot:'Reboot after provision', c_skip_root:'Do not set root password (leave empty)', l_new_root:'New root password (LuCI)', edge_root_help:'Required for LuCI. SSH uses operator key; password SSH is disabled after provision.', edge_guest_help:'Isolated guest, ISP direct (not VPN). Default: visible SSID (join-QR). Optional hidden. PSK=Wi‑Fi password; desk PIN=staff grant.', l_guest_visible:'SSID visibility', opt_hidden:'Hidden', opt_visible:'Visible (default)', c_edge_offline:'Offline deploy', btn_edge_offline_prep:'Prepare offline pack', edge_offline_help:'Uses local pack (token + agents). No SSH to primary at deploy.', edge_offline_prep_warn:'Needs internet + SSH primary now. Re-run after upgrades or token rotate.', btn_edge_probe:'Probe device →', edge_h_wifi:'Wi‑Fi AP', edge_wifi_hint:'You can fill 2.4 only — empty 5 GHz SSID/key is copied from 2.4 (and vice versa).', edge_h_guest:'Guest Wi‑Fi', edge_guest_detail_hint:'Isolated SSID; clients use ISP (not router VLESS). PSK = Wi‑Fi password; desk PIN = staff grant.', edge_wan_hint:'Uplink to ISP. Optional DNS e.g. 1.1.1.1 9.9.9.9.', deploy_log_title:'Deploy log', jobs_title:'Operator jobs', jobs_note:'Live on netductor-op serve. Cancel is cooperative (current SSH step may finish).', jobs_refresh:'Refresh', jobs_running:'Running only', jobs_empty:'(no jobs)', jobs_cancel:'Cancel', jobs_cancel_confirm:'Cancel job?', edge_run_working:'Working… job steps below (survives refresh)', edge_run_done:'Finished — see log', edge_run_err:'Failed — see log', netmap_title:'Network map', netmap_note:'Canonical CIDRs — SR and edge LAN.', edge_h_lan:'LAN', edge_lan_hint:'Shops: prefer 10.120.<N>.0/24 (Settings → Network map). Not 10.9.8 or 10.88.', edge_mod_agent:'Agent install', edge_mod_lan:'LAN baseline', edge_mod_wan:'WAN / uplink', edge_mod_wifi:'Wi‑Fi AP', edge_mod_guest:'Guest Wi‑Fi', edge_mod_vpn:'VPN client (VLESS)', edge_mod_fs:'Expand root FS (SD/mmc)', edge_mod_overlay:'External overlay / extroot', edge_mod_usb_format:'Format USB (destroy data)', edge_storage_title:'Edge storage (USB)', edge_storage_help:'Day-2: backup A+B, mute alerts, restore from primary.', btn_st_list:'List backups', btn_st_list:'List backups', btn_st_backup:'Backup now (A+B)', btn_st_restore:'Restore from primary', btn_st_restore_fmt:'Format USB + restore', btn_st_mute:'Mute USB alerts', btn_st_unmute:'Unmute USB alerts', btn_st_offline:'Build offline pack (primary)', edge_mod_usb_format_help:'Wipe selected USB and create ext4 for extroot. Irreversible.', edge_mod_ssh:'SSH harden', edge_mod_vpn_help:'Router joins VLESS → secondary/primary. LAN clients use this path; guest Wi‑Fi stays ISP.', edge_mod_help:'Toggle modules for this device. Grey = not possible on this hardware.', edge_stat_mem:'Memory free', edge_stat_storage:'Storage', edge_stat_expand:'Expand hint', edge_chip_eth:'Eth ×', edge_chip_wan_yes:'WAN capable', edge_chip_wan_no:'No WAN', edge_chip_no_radio:'No radio', edge_yes:'yes', edge_no:'no', edge_storage_flash:'flash', edge_storage_overlay:'overlay', edge_storage_root:'root', edge_ph_router:'e.g. 192.168.1.1 — not filled automatically', btn_edge_to_modules:'Continue →', btn_edge_to_details:'Continue →', btn_edge_replan:'Re-probe / replan', btn_back:'Back', btn_edge_details:'Details →', btn_edge_dry:'Dry-run plan', btn_edge_modules:'Modules →', c_advanced:'Advanced modules', edge_details_help:'Fill only modules you enabled.', edge_h_card:'Device card', edge_h_connect:'Connect', edge_h_details:'Details', edge_h_modules:'Modules', edge_mod_help:'Toggle modules for this device class.', l_action:'Action', l_display:'Display name', l_key_pass:'Key passphrase', l_label:'Label', l_preset:'Preset', l_primary:'Primary host', l_router:'Router host', l_router_pass:'Router password', l_rpi_id:'Device id', l_rpi_lan:'Router LAN IP', l_server:'Agent server URL', edge_note:'OpenWrt edge: agent then UCI. Primary SSH uses port 52222 (after harden); router SSH port 22. Prefill from Fleet/Settings.', edge_net_help:'Only if Configure network is on.', l_lan_mask:'LAN mask', l_dhcp_start:'DHCP start', l_dhcp_limit:'DHCP limit', l_ssid24:'Wi-Fi SSID 2.4', l_key24:'Wi-Fi key 2.4', l_ssid5:'Wi-Fi SSID 5', l_key5:'Wi-Fi key 5', l_wan_ip:'WAN IP', l_wan_mask:'WAN mask', l_wan_gw:'WAN gateway', l_wan_dns:'WAN DNS', l_wan_dns_help:'Optional. Space or comma. Recommended 1.1.1.1 9.9.9.9 (CF first). Empty=ISP. Not VPN dns mode.', l_pppoe_user:'PPPoE user', l_pppoe_pass:'PPPoE password', l_guest_ssid:'Guest SSID', l_guest_pin:'Guest desk PIN', l_guest_pin_help:'Staff PIN to grant internet — not the Wi‑Fi password.', l_guest_psk:'Guest PSK (Wi‑Fi)', l_guest_psk_help:'Wi‑Fi password embedded in join QR. Required (no open network).', c_push:'Push RSC + harden',
  tg_note:'Telegram installs on primary only (token + numeric admin id required).',
  build_note:'CI build-host from Mac (op). PB/BP tunnels. Not VPN exit.', l_build_host:'Build host', l_build_pass:'Build password', l_build_label:'Build label', btn_fleet:'Fleet deploy', btn_primary:'Deploy primary', btn_build:'Deploy build-host', btn_secondary:'Deploy secondary',
  btn_edge:'Provision edge', h_luci:'LuCI (LAN)', luci_help:'Temporarily enable OpenWrt LuCI web UI with auto-off timer (default 1h). Use while on the same LAN as the router. SSH via = operator key from this Mac; Agent via = command queued on primary (router needs agent online).', l_device_id:'Device id (agent)', l_device_id_help:'Edge device_id from provision (e.g. site-home1). Required for Via=Agent; optional for SSH if you only use key+host.', l_luci_action:'Action', l_luci_hours:'Hours (TTL then LuCI off)', l_luci_via:'Via', btn_luci:'LuCI control', btn_site:'Save / push site', btn_creds:'Collect',
  btn_tunnel_start:'Start / ensure', btn_tunnel_stop:'Stop', btn_session_issue:'Issue session', btn_session_load:'Load session file',
  btn_raw:'Raw JSON', btn_send:'Send',
  set_conn:'Connection', set_phost:'Primary host (public)', set_vpn:'Primary VPN host', set_key:'SSH key', set_user:'SSH user', set_prefer:'Prefer VPN for SSH',
  set_api_port:'API local port', set_api_base:'API base', set_sec_host:'Secondary host', set_session:'Node session (sessionStorage)',
  set_session_note:'Not stored in localStorage', set_disk_note:'Save writes ~/.config/netductor/operator.conf (shared with TUI). Empty fields get values from file on load.',
  save:'Save', set_gh:'GitHub token (on primary)', set_gh_note:'Stored on node as secrets/github_token. Avoids release-list 403.', set_gh_tok:'Token', set_gh_st:'Status', set_gh_save:'Save on primary', set_gh_clear:'Clear', set_gh_refresh:'Refresh status', ctrl_note:'Day-2 via node API. Tunnel: direct → VPN SSH → public SSH. Session kept in sessionStorage only (not localStorage).',
  result:'Result', tunnel:'Tunnel', idle:'idle',
  sec_overview:'Overview', sec_vpn:'VPN', sec_nodes:'Nodes', sec_edge:'Edge', sec_nvr:'NVR', sec_git:'Git / Reg', sec_backup:'Backup', sec_dns:'DNS', sec_probes:'Probes', sec_adv:'Advanced',
  adv_note:'Generic POST/GET for any session API path', l_method:'method', l_path:'path', l_body:'JSON body',
  b_health:'Health', b_stack:'Stack', b_stack_apply:'Stack apply', b_stack_rb:'Stack rollback', b_stack_wd:'Stack WD', b_doctor:'Doctor', b_domain:'Domain', b_bot:'Bot status', b_status:'Status', b_overview:'Fleet overview', b_metrics:'Metrics',
  b_metrics_hist:'Metrics history', b_addons:'Addons', b_lampac:'Lampac', b_sni:'SNI', b_sni_presets:'SNI presets',
  b_latest:'Latest', b_sessions:'Sessions', b_gh_token:'GitHub token', b_upd:'Update status', b_rel_local:'Local releases', b_rel_tags:'Git tags', b_rel_mirror:'Mirror fetch', b_rel_build:'Build local', b_vpn_users:'List users', b_vpn_refresh:'Refresh links',
  b_vpn_add:'Add user', b_vpn_enable:'Enable', b_vpn_disable:'Disable', b_vpn_revoke:'Revoke', b_vpn_link:'Get links',
  l_vpn_policy:'Access policy', l_vpn_policy_help:'User or edge: load, tick services, save.', l_pol_internet:'Internet egress', l_pol_all:'All internal services', b_vpn_pol_load:'Load user policy', b_vpn_pol_save:'Save user policy', b_vpn_pol_media:'Preset: media', b_vpn_pol_full:'Preset: full', b_vpn_pol_none:'Preset: none', l_edge_pol:'Edge device id', b_edge_pol_load:'Load edge policy', b_edge_pol_save:'Save edge policy',
  l_dns_id:'list id', l_dns_en:'enabled', b_dns_set:'Set list', b_dns_on:'on', b_dns_off:'off',
  l_bak_hour:'hour (0-23)', l_bak_min:'minute (0-59)', l_bak_tz:'timezone', b_bak_save:'Save schedule',
  l_git_repo:'repo', l_git_pipe:'pipeline', l_git_rev:'rev / path', l_git_art:'artifact path',
  l_git_proj:'Projects (mirror)', l_git_proj_name:'name', l_git_proj_up:'upstream (org/repo)', l_git_proj_wf:'workflow path', l_git_proj_pipe:'pipeline',
  b_git_run:'Run pipeline', b_git_init:'Init repo', b_git_log:'Log', b_git_show:'Show', b_git_art:'Get artifact',
  b_git_proj_list:'List projects', b_git_proj_add:'Add project', b_git_proj_sync:'Sync', b_git_proj_build:'Build', b_git_proj_detail:'Project card', b_git_proj_migrate:'Migrate from GH', b_git_proj_sync_all:'Sync all', b_git_proj_build_all:'Build all (vps)', b_git_proj_vps_queue:'VPS queue', b_git_proj_vps_cancel:'Cancel VPS job', b_git_proj_mirror_all:'Mirror→GH all', l_git_mig_owner:'migrate owner', l_git_mig_mode:'mode',
  b_nodes:'Nodes', b_self:'Self', b_sec_st:'Secondary status', b_sec_links:'Secondary links',
  b_ssh_hosts:'SSH hosts', b_ssh_clear:'SSH hosts clear', b_mtls:'mTLS certs', b_sites:'Sites',
  b_hostname:'Set hostname', b_svc_restart:'Restart service', b_journal:'Journal',
  l_vpn_name:'name', l_vpn_note:'note', l_vpn_act:'user action', l_hostname:'hostname', l_node_id:'node id',
  l_svc:'service restart', l_journal:'journal unit', yes:'yes', no:'no',
  nvr_note:'Day-2 NVR via node session (tunnel + session). Same ops as TUI wizard.',
  l_nvr_action:'Action', l_edge_id:'Edge device id', l_cam_name:'Camera name/id', l_cam_ip:'Camera LAN IP', l_cam_pass:'Camera password',
  btn_nvr:'Run NVR', c_adv:'Show Advanced (full session API)', sub_nvr:'NVR',
  ssh_title:'SSH allowlist', ssh_note:'Inventory → fingerprints → Save (no live change) / Merge / Enforce. Probe secondary first.', ssh_inv:'Inventory', ssh_probe:'Probe secondary', ssh_status:'Allowlist status', ssh_fps:'Fingerprints (one per line)', ssh_save:'Save allowlist', ssh_merge:'Merge', ssh_enf:'Enforce + secondary',
  reboot_title:'Reboot (challenge)', reboot_note:'Request code → enter correct code within 3 min. Primary: shutdown +1min. Secondary: agent queue.', reboot_target:'Target (primary or secondary id)', reboot_challenge:'Get code', reboot_code:'Code', reboot_confirm:'Confirm reboot'
},
ru:{
  app_title:'netductor-op', nav_installer:'Деплой', nav_control:'Управление', nav_settings:'Настройки', meta_line:'Клиент Mac · API ноды', nav_grp_fleet:'Флот', nav_servers:'Серверы', nav_sites:'Сайты', servers_note:'Узлы control plane. Клик по карточке — детали и действия.', sites_note:'Edge-сайты — топология, зоны, метрики. «Добавить сайт» открывает мастер OpenWrt.', sites_add:'+ Добавить сайт', sites_empty:'Пока нет сайтов. Добавьте сайт через мастер OpenWrt.', sites_offline:'Нужны туннель и session, чтобы загрузить сайты с primary.', card_openwrt:'OpenWrt / сайт', card_openwrt_d:'Probe → модули → provision', card_mt_d:'RSC push / site shell', actions:'Действия', nav_grp_deploy:'Деплой', nav_grp_operate:'Управление', nav_collapse:'Свернуть', btn_st_list:'Список бэкапов', l_port:'Порт', b_gh_token:'Токен GitHub',
  tab_installer:'Установка', tab_control:'Управление', tab_settings:'Настройки', tab_updates:'Обновления',
  upd_local:'Локально', upd_latest:'Latest', upd_pick:'Релиз', upd_apply:'Применить на primary', upd_refresh:'Обновить', upd_available:'есть обновление', upd_ok:'актуально',
  sub_fleet:'Флот', sub_primary:'Primary', sub_build:'Build-host', sub_secondary:'Secondary', sub_edge:'OpenWrt', sub_site:'MikroTik', sub_creds:'Учётные данные',
  inst_note:'Деплой с этого Mac по SSH.',
  l_phost:'Хост primary', l_ppass:'Пароль primary', l_shost:'Хост secondary', l_spass:'Пароль secondary',
  l_domain:'Метка DOMAIN (опц., не создаёт имена)', l_domain_primary:'Домен primary', l_domain_vpn:'Домен secondary', l_domain_redirect:'URL redirect', l_ssh_port:'Порт SSH', l_redir_port:'HTTPS redirect', l_mtls_port:'Agent mTLS', l_lampac_port:'Lampac', l_le:'Email LE', l_sni:'SNI (маскировка Reality)', l_key:'SSH-ключ', l_host:'Хост', l_pass:'Пароль',
  l_pkey:'Ключ primary', l_user:'Пользователь', l_arch:'Архитектура agent', l_server_url:'URL сервера (mTLS)', l_keypass:'Passphrase ключа',
  l_lan:'LAN IP (опц.)', l_wan:'WAN proto', l_ssid:'Wi-Fi SSID', l_wifikey:'Ключ Wi-Fi',
  l_site_id:'ID сайта', l_name:'Имя', l_rpi:'ID RPi', l_mtid:'ID MikroTik', l_rpilan:'Шлюз LAN RPi',
  l_mthost:'Хост MT', l_mtuser:'Пользователь MT', l_mtpass:'Пароль MT', l_mtport:'Порт MT', l_opkey:'Ключ оператора',
  l_role:'Роль', l_tg_token:'Токен TG-бота', l_tg_admin:'TG admin id',
  c_do_primary:'Primary', c_do_secondary:'Secondary', c_do_build:'Build-host', c_lampac:'Lampac', c_git:'Git', c_tg:'Telegram-бот', 
  c_netcfg:'Настроить сеть', c_guest:'Гостевой Wi-Fi', c_reboot:'Перезагрузка после provision', c_skip_root:'Не задавать пароль root (оставить пустым)', l_new_root:'Новый пароль root (LuCI)', edge_root_help:'Нужен для LuCI. SSH — ключ оператора; password SSH после provision отключается.', edge_guest_help:'Гостевая изолирована, интернет через провайдера (не VPN). По умолчанию: видимый SSID (join-QR). Скрытый — опция. PSK=пароль Wi‑Fi; PIN=выдача доступа персоналом.', l_guest_visible:'Видимость SSID', opt_hidden:'Скрытый', opt_visible:'Видимый (по умолчанию)', c_edge_offline:'Офлайн-деплой', btn_edge_offline_prep:'Подготовить offline-пакет', edge_offline_help:'Пакет с диска (token + agents). Без SSH на primary при деплое.', edge_offline_prep_warn:'Нужен интернет и SSH на primary сейчас. Повторить после апгрейда или смены token.', btn_edge_probe:'Опросить устройство →', edge_h_wifi:'Wi‑Fi AP', edge_wifi_hint:'Можно заполнить только 2.4 — пустые SSID/ключ 5 ГГц копируются с 2.4 (и наоборот).', edge_h_guest:'Гостевой Wi‑Fi', edge_guest_detail_hint:'Отдельный SSID; гости в ISP (не через VLESS роутера). PSK = пароль Wi‑Fi; PIN = выдача персоналом.', edge_wan_hint:'Аплинк в ISP. DNS опционально, напр. 1.1.1.1 9.9.9.9.', deploy_log_title:'Лог деплоя', jobs_title:'Задачи op', jobs_note:'Живут в netductor-op serve. Cancel — кооперативный (текущий SSH-шаг может доработать).', jobs_refresh:'Обновить', jobs_running:'Только running', jobs_empty:'(нет задач)', jobs_cancel:'Отменить', jobs_cancel_confirm:'Отменить задачу?', edge_run_working:'Идёт применение… шаги в логе ниже', edge_run_done:'Готово — смотрите лог', edge_run_err:'Ошибка — смотрите лог', netmap_title:'Карта сетей', netmap_note:'Канонические CIDR — SR и edge LAN.', edge_h_lan:'LAN', edge_lan_hint:'Магазины: лучше 10.120.<N>.0/24 (Настройки → Карта сетей). Не 10.9.8 и не 10.88.', edge_mod_agent:'Установка agent', edge_mod_lan:'LAN baseline', edge_mod_wan:'WAN / uplink', edge_mod_wifi:'Wi‑Fi AP', edge_mod_guest:'Гостевой Wi‑Fi', edge_mod_vpn:'VPN‑клиент (VLESS)', edge_mod_fs:'Расширить root (SD/mmc)', edge_mod_overlay:'Внешний overlay / extroot', edge_mod_usb_format:'Форматировать USB (данные будут уничтожены)', edge_storage_title:'Накопитель edge (USB)', edge_storage_help:'Day-2: бэкап A+B, mute алертов, restore с primary.', btn_st_backup:'Бэкап сейчас (A+B)', btn_st_restore:'Restore с primary', btn_st_restore_fmt:'Format USB + restore', btn_st_mute:'Mute USB-алертов', btn_st_unmute:'Включить USB-алерты', btn_st_offline:'Собрать offline pack (на primary)', edge_mod_usb_format_help:'Стереть USB и создать ext4 под extroot. Необратимо.', edge_mod_ssh:'SSH harden', edge_mod_vpn_help:'Роутер поднимает VLESS → secondary/primary. Клиенты LAN идут через VPN; гости — напрямую в ISP.', edge_mod_help:'Модули для этого устройства. Серые — нельзя на этом железе.', edge_stat_mem:'Свободная память', edge_stat_storage:'Накопитель', edge_stat_expand:'Expand root', edge_chip_eth:'Eth ×', edge_chip_wan_yes:'есть WAN', edge_chip_wan_no:'без WAN', edge_chip_no_radio:'нет radio', edge_yes:'да', edge_no:'нет', edge_storage_flash:'flash', edge_storage_overlay:'overlay', edge_storage_root:'root', edge_ph_router:'например 192.168.1.1 — само не подставляется', btn_edge_to_modules:'Модули →', btn_edge_to_details:'Параметры →', btn_edge_replan:'Повторить probe', btn_back:'Назад', btn_edge_details:'Параметры →', btn_edge_dry:'План (dry-run)', btn_edge_modules:'Модули →', c_advanced:'Advanced-модули', edge_details_help:'Заполните только включённые модули.', edge_h_card:'Карточка устройства', edge_h_connect:'Подключение', edge_h_details:'Параметры', edge_h_modules:'Модули', edge_mod_help:'Модули для этого класса устройства.', l_action:'Действие', l_display:'Отображаемое имя', l_key_pass:'Passphrase ключа', l_label:'Метка', l_preset:'Пресет', l_primary:'Primary host', l_router:'Хост роутера', l_router_pass:'Пароль роутера', l_rpi_id:'ID устройства', l_rpi_lan:'LAN IP роутера', l_server:'URL agent server', edge_note:'OpenWrt edge: агент, затем UCI. SSH на primary — порт 52222 (после harden); на роутер — 22. Prefill из Fleet/Settings.', edge_net_help:'Только если включено «Настроить сеть».', l_lan_mask:'Маска LAN', l_dhcp_start:'DHCP start', l_dhcp_limit:'DHCP limit', l_ssid24:'SSID 2.4', l_key24:'Ключ 2.4', l_ssid5:'SSID 5', l_key5:'Ключ 5', l_wan_ip:'WAN IP', l_wan_mask:'Маска WAN', l_wan_gw:'Шлюз WAN', l_wan_dns:'DNS WAN', l_wan_dns_help:'Опционально. Пробел или запятая. Рекомендуем 1.1.1.1 9.9.9.9 (сначала CF). Пусто=DNS провайдера. Не режим dns=vpn у edge.', l_pppoe_user:'PPPoE user', l_pppoe_pass:'PPPoE password', l_guest_ssid:'Guest SSID', l_guest_pin:'PIN продавца', l_guest_pin_help:'PIN персонала для выдачи интернета — не пароль Wi‑Fi.', l_guest_psk:'Guest PSK (Wi‑Fi)', l_guest_psk_help:'Пароль Wi‑Fi в join-QR. Обязателен (open-сеть не поддерживается).', c_push:'Push RSC + harden',
  tg_note:'Telegram ставится только на primary (нужны token и числовой admin id).',
  build_note:'CI build-host с Mac (op). Туннели PB/BP к primary. Не VPN-exit.', l_build_host:'Хост build', l_build_pass:'Пароль build', l_build_label:'Метка build', btn_fleet:'Деплой флота', btn_primary:'Деплой primary', btn_build:'Деплой build-host', btn_secondary:'Деплой secondary',
  btn_edge:'Поставить edge', h_luci:'LuCI (LAN)', luci_help:'Временно включить веб-UI LuCI OpenWrt с таймером автовыключения (по умолчанию 1ч). Нужен доступ в ту же LAN, что и роутер. Via=SSH — ключ оператора с Mac; Via=Agent — команда в очередь primary (агент на роутере должен быть online).', l_device_id:'Device id (agent)', l_device_id_help:'ID edge с provision (напр. site-home1). Обязателен для Via=Agent; для SSH можно только host+ключ.', l_luci_action:'Действие', l_luci_hours:'Часы (TTL, потом LuCI выкл)', l_luci_via:'Канал', btn_luci:'Управление LuCI', btn_site:'Сохранить / push сайта', btn_creds:'Собрать',
  btn_tunnel_start:'Старт / ensure', btn_tunnel_stop:'Стоп', btn_session_issue:'Выдать session', btn_session_load:'Загрузить session',
  btn_raw:'Сырой JSON', btn_send:'Отправить',
  set_conn:'Подключение', set_phost:'Primary (публичный)', set_vpn:'Primary через VPN', set_key:'SSH-ключ', set_user:'Пользователь SSH', set_prefer:'Сначала VPN для SSH',
  set_api_port:'Локальный порт API', set_api_base:'База API', set_sec_host:'Хост secondary', set_session:'Session ноды (sessionStorage)',
  set_session_note:'Не хранится в localStorage', set_disk_note:'Save пишет ~/.config/netductor/operator.conf (общий с TUI). Пустые поля подставляются из файла.',
  save:'Сохранить', set_gh:'GitHub token (на primary)', set_gh_note:'Файл secrets/github_token на ноде. Снимает 403 при списке релизов.', set_gh_tok:'Токен', set_gh_st:'Статус', set_gh_save:'Сохранить на primary', set_gh_clear:'Удалить', set_gh_refresh:'Обновить статус', ctrl_note:'Day-2 через API ноды. Туннель: direct → VPN SSH → public SSH. Session только в sessionStorage.',
  result:'Результат', tunnel:'Туннель', idle:'ожидание',
  sec_overview:'Обзор', sec_vpn:'VPN', sec_nodes:'Ноды', sec_edge:'Edge', sec_nvr:'NVR', sec_git:'Git / Registry', sec_backup:'Бэкап', sec_dns:'DNS', sec_probes:'Probes', sec_adv:'Дополнительно',
  adv_note:'Произвольный POST/GET к session API', l_method:'метод', l_path:'путь', l_body:'JSON body',
  b_health:'Health', b_stack:'Стек', b_stack_apply:'Применить стек', b_stack_rb:'Откат стека', b_stack_wd:'Watchdog', b_doctor:'Doctor', b_domain:'Домен', b_bot:'Статус бота', b_status:'Статус', b_overview:'Обзор флота', b_metrics:'Метрики',
  b_metrics_hist:'История метрик', b_addons:'Дополнения', b_lampac:'Lampac', b_sni:'SNI', b_sni_presets:'Пресеты SNI',
  b_latest:'Latest', b_sessions:'Sessions', b_vpn_users:'Список users', b_vpn_refresh:'Обновить ссылки',
  b_vpn_add:'Добавить user', b_vpn_enable:'Включить', b_vpn_disable:'Выключить', b_vpn_revoke:'Отозвать', b_vpn_link:'Ссылки',
  l_vpn_policy:'Политика доступа', l_vpn_policy_help:'User или edge: загрузить, галочки, сохранить.', l_pol_internet:'Выход в интернет', l_pol_all:'Все внутренние сервисы', b_vpn_pol_load:'Загрузить user', b_vpn_pol_save:'Сохранить user', b_vpn_pol_media:'Пресет: media', b_vpn_pol_full:'Пресет: full', b_vpn_pol_none:'Пресет: none', l_edge_pol:'ID edge', b_edge_pol_load:'Загрузить edge', b_edge_pol_save:'Сохранить edge',
  l_dns_id:'id списка', l_dns_en:'вкл', b_dns_set:'Применить', b_dns_on:'вкл', b_dns_off:'выкл',
  l_bak_hour:'час (0-23)', l_bak_min:'мин (0-59)', l_bak_tz:'часовой пояс', b_bak_save:'Сохранить расписание',
  l_git_repo:'репо', l_git_pipe:'pipeline', l_git_rev:'rev / path', l_git_art:'путь артефакта',
  l_git_proj:'Проекты (mirror)', l_git_proj_name:'имя', l_git_proj_up:'upstream (org/repo)', l_git_proj_wf:'путь workflow', l_git_proj_pipe:'pipeline',
  b_git_run:'Запуск pipeline', b_git_init:'Init репо', b_git_log:'Log', b_git_show:'Show', b_git_art:'Артефакт',
  b_git_proj_list:'Список проектов', b_git_proj_add:'Добавить проект', b_git_proj_sync:'Sync', b_git_proj_build:'Build', b_git_proj_detail:'Карточка проекта', b_git_proj_migrate:'Миграция с GH', b_git_proj_sync_all:'Sync все', b_git_proj_build_all:'Build все (vps)', b_git_proj_vps_queue:'Очередь VPS', b_git_proj_vps_cancel:'Отменить VPS job', b_git_proj_mirror_all:'Mirror→GH все', l_git_mig_owner:'владелец миграции', l_git_mig_mode:'режим',
  b_nodes:'Ноды', b_self:'Self', b_sec_st:'Статус secondary', b_sec_links:'Ссылки secondary',
  b_ssh_hosts:'SSH hosts', b_ssh_clear:'Очистить SSH hosts', b_mtls:'mTLS сертификаты', b_sites:'Сайты', b_upd:'Статус обновлений', b_rel_local:'Локальные релизы', b_rel_tags:'Git теги', b_rel_mirror:'Mirror fetch', b_rel_build:'Сборка local',
  b_hostname:'Задать hostname', b_svc_restart:'Restart сервиса', b_journal:'Journal',
  l_vpn_name:'имя', l_vpn_note:'заметка', l_vpn_act:'user', l_hostname:'hostname', l_node_id:'id ноды',
  l_svc:'restart сервиса', l_journal:'journal unit', yes:'да', no:'нет',
  nvr_note:'NVR day-2 через session (туннель + session). Те же операции, что TUI.',
  l_nvr_action:'Операция', l_edge_id:'Edge device id', l_cam_name:'Имя/id камеры', l_cam_ip:'LAN IP камеры', l_cam_pass:'Пароль камеры',
  btn_nvr:'Выполнить NVR', c_adv:'Показать Advanced (полный session API)', ssh_title:'SSH allowlist', ssh_note:'Список ключей → fingerprints → Сохранить (без смены live) / Merge / Enforce. Сначала опрос secondary.', ssh_inv:'Список ключей', ssh_probe:'Опрос secondary', ssh_status:'Статус allowlist', ssh_fps:'Fingerprints (по строке)', ssh_save:'Сохранить allowlist', ssh_merge:'Добавить (merge)', ssh_enf:'Enforce + secondary', reboot_title:'Перезагрузка (код)', reboot_note:'Запросить код → ввести верный за 3 мин. Primary: shutdown +1мин. Secondary: очередь агента.', reboot_target:'Цель (primary или id secondary)', reboot_challenge:'Получить код', reboot_code:'Код', reboot_confirm:'Подтвердить reboot', sub_nvr:'NVR'
}};
function t(k){ const d=I18N[uiLang]||I18N.en; return (d&&d[k])||(I18N.en[k])||k; }
function detectUiLang(){
  const saved=localStorage.getItem('nd_op_lang');
  if(saved==='ru'||saved==='en') return saved;
  const langs=[navigator.language||''].concat(navigator.languages||[]);
  for(const L of langs){ if(String(L).toLowerCase().startsWith('ru')) return 'ru'; }
  return 'en';
}
let uiLang=detectUiLang();
function applyAdvancedVisibility(){
  const show = !!(settings().show_advanced === true || settings().show_advanced === '1' || settings().show_advanced === 1);
  document.querySelectorAll('#controlSub button[data-csec="adv"]').forEach(b=>{ b.style.display = show ? '' : 'none'; });
  const adv = document.getElementById('csec-adv');
  if(adv && !show) adv.style.display = 'none';
}
function applyI18n(){
  applyAdvancedVisibility();

  const d=I18N[uiLang]||I18N.en;
  document.documentElement.lang = uiLang==='ru'?'ru':'en';
  document.querySelectorAll('[data-i18n-placeholder]').forEach(el=>{ const k=el.getAttribute('data-i18n-placeholder'); const v=t(k); if(v&&v!==k) el.setAttribute('placeholder', v); });
  document.querySelectorAll('[data-i18n]').forEach(el=>{ const k=el.getAttribute('data-i18n'); if(d[k]) el.textContent=d[k]; });
  document.querySelectorAll('[data-i18n-html]').forEach(el=>{ const k=el.getAttribute('data-i18n-html'); if(d[k]) el.innerHTML=d[k]; });
  document.querySelectorAll('#mainTabs button').forEach(b=>{
    if(b.dataset.main==='installer') b.textContent=d.tab_installer||b.textContent;
    if(b.dataset.main==='control') b.textContent=d.tab_control||b.textContent;
    if(b.dataset.main==='settings') b.textContent=d.tab_settings||b.textContent;
  });
  document.querySelectorAll('#subInstaller button').forEach(b=>{
    const map={fleet:'sub_fleet',primary:'sub_primary',build:'sub_build',secondary:'sub_secondary',edge:'sub_edge',site:'sub_site',nvr:'sub_nvr',creds:'sub_creds'};
    const k=map[b.dataset.tab]; if(k&&d[k]) b.textContent=d[k];
  });
  document.querySelectorAll('#controlSub button').forEach(b=>{
    const k='sec_'+b.dataset.csec; if(d[k]) b.textContent=d[k];
  });
  const le=document.getElementById('langEn'), lr=document.getElementById('langRu');
  if(le) le.classList.toggle('active', uiLang==='en');
  if(lr) lr.classList.toggle('active', uiLang==='ru');
  try{ loadCatalogButtons().then(()=>mountButtons()).catch(()=>mountButtons()); }catch(e){}
  try{ wirePasswordToggles(); }catch(e){}
}


function showMain(name){
  // .hidden uses display:none !important — must toggle class, not only style.display
  const setVis = (id, on, display)=>{
    const el=document.getElementById(id); if(!el) return;
    el.classList.toggle('hidden', !on);
    el.style.display = on ? display : 'none';
  };
  setVis('main-installer', name==='installer', 'block');
  setVis('main-control', name==='control', 'block');
  setVis('main-settings', name==='settings', 'block');
  setVis('subInstaller', name==='installer', 'flex');
  setVis('deployLogSec', name==='installer', 'block');
  document.querySelectorAll('#mainTabs button').forEach(b=>b.classList.toggle('active', b.dataset.main===name));
  if(name==='control'){
    updateTunnelHint();
    try{ fillUpdateUI(); }catch(e){}
    try{ wirePasswordToggles(); }catch(e){}
    ensureTunnel().then(()=>{ refreshTunnelBadge(); fillUpdateUI(); });
  }
  if(name==='settings') loadSettingsForm();
  document.querySelectorAll('.fleet-page').forEach(p=>p.classList.add('hidden'));
  try{ syncSideNav(); }catch(e){}
}

document.querySelectorAll('#mainTabs button').forEach(b=>b.onclick=()=>showMain(b.dataset.main));
document.querySelectorAll('#subInstaller button').forEach(btn=>{
  btn.onclick=()=>{
    document.querySelectorAll('#subInstaller button').forEach(b=>b.classList.remove('active'));
    document.querySelectorAll('#main-installer .panel').forEach(p=>p.classList.remove('active'));
    btn.classList.add('active');
    document.getElementById('panel-'+btn.dataset.tab).classList.add('active');
  };
});
document.querySelectorAll('#controlSub button').forEach(btn=>{
  btn.onclick=()=>{
    document.querySelectorAll('#controlSub button').forEach(b=>b.classList.remove('active'));
    btn.classList.add('active');
    document.querySelectorAll('.csec').forEach(p=>p.style.display='none');
    document.getElementById('csec-'+btn.dataset.csec).style.display='block';
  };
});

const logEl=document.getElementById('log'), stepsEl=document.getElementById('steps');
function line(s){ logEl.textContent+=s+(s.endsWith('\n')?'':'\n'); logEl.scrollTop=logEl.scrollHeight; }
function resetLog(){ logEl.textContent=''; stepsEl.innerHTML=''; }
function chip(id,st){ let el=stepsEl.querySelector('[data-id="'+id+'"]'); if(!el){el=document.createElement('span');el.dataset.id=id;el.textContent=id;stepsEl.appendChild(el);} el.className='chip '+(st||''); }
function parseSteps(chunk){ chunk.split('\n').forEach(l=>{ const m=l.match(/^step (\S+)/); if(!m)return; if(l.includes(' ERROR'))chip(m[1],'err'); else if(l.includes(' done'))chip(m[1],'ok'); else chip(m[1],'run'); }); }
function setEdgeRunStatus(kind, msg){
  const el=document.getElementById('edgeRunStatus');
  if(!el) return;
  if(!msg){ el.hidden=true; el.textContent=''; el.className='edge-run-status'; return; }
  el.hidden=false;
  el.className='edge-run-status edge-run-'+kind;
  el.textContent=msg;
}

/** Thin UI: start job on serve, poll until terminal. Survives page refresh if job_id in sessionStorage. */
async function jobStart(type, params){
  const res=await fetch('/v1/jobs',{method:'POST',headers:{'Content-Type':'application/json','X-Netductor-Token':ND_TOKEN},body:JSON.stringify({type,params})});
  const data=await res.json().catch(()=>({}));
  if(!res.ok) throw new Error(data.error||('job start HTTP '+res.status));
  try{ sessionStorage.setItem('nd_op_job_'+type, data.id); }catch(e){}
  return data;
}
async function jobGet(id){
  const res=await fetch('/v1/jobs/'+encodeURIComponent(id),{headers:{'X-Netductor-Token':ND_TOKEN}});
  const data=await res.json().catch(()=>({}));
  if(!res.ok) throw new Error(data.error||('job get HTTP '+res.status));
  return data;
}
async function jobWait(id, onUpdate){
  const terminal={succeeded:1,failed:1,cancelled:1,interrupted:1};
  for(;;){
    const j=await jobGet(id);
    if(onUpdate) onUpdate(j);
    if(terminal[j.status]) return j;
    await new Promise(r=>setTimeout(r,1000));
  }
}

async function jobCancel(id){
  const res=await fetch('/v1/jobs/'+encodeURIComponent(id)+'/cancel',{method:'POST',headers:{'X-Netductor-Token':ND_TOKEN}});
  const data=await res.json().catch(()=>({}));
  if(!res.ok) throw new Error(data.error||('cancel HTTP '+res.status));
  return data;
}
let jobsShowRunningOnly=false;
async function refreshJobsPanel(){
  const el=document.getElementById('jobsList');
  if(!el) return;
  try{
    const q=jobsShowRunningOnly?'?status=running':'';
    const res=await fetch('/v1/jobs'+q,{headers:{'X-Netductor-Token':ND_TOKEN}});
    const data=await res.json().catch(()=>({}));
    if(!res.ok){ el.innerHTML='<span class="note">'+escHtml(data.error||('HTTP '+res.status))+'</span>'; return; }
    const list=data.jobs||[];
    const running=(list.filter(j=>j.status==='running'||j.status==='queued')).length;
    const cnt=document.getElementById('jobsRunningCount');
    const hb=document.getElementById('btnHeaderJobs');
    if(cnt){ cnt.textContent=String(running); cnt.classList.toggle('hide', running<=0); }
    if(hb){ hb.classList.toggle('has-run', running>0); }
    const sb=document.getElementById('sbJobsN'); if(sb) sb.textContent=String(running);

    if(!list.length){ el.innerHTML='<span class="note">'+(typeof t==='function'?t('jobs_empty'):'(no jobs)')+'</span>'; return; }
    el.innerHTML='';
    list.forEach(j=>{
      const row=document.createElement('div');
      row.className='job-row chip-row';
      row.style.cssText='display:flex;flex-wrap:wrap;align-items:center;gap:0.4rem;margin:0.25rem 0;padding:0.35rem 0.5rem;border:1px solid var(--border,#333);border-radius:6px';
      const st=j.status||'';
      const badge=document.createElement('span');
      badge.className='chip '+(st==='running'||st==='queued'?'run':(st==='failed'||st==='cancelled'?'err':'ok'));
      badge.textContent=st;
      const info=document.createElement('span');
      info.style.flex='1';
      info.textContent=(j.type||'')+' · '+(j.id||'').slice(0,12);
      if(j.error) info.title=j.error;
      row.appendChild(badge); row.appendChild(info);
      if(st==='running'||st==='queued'){
        const btn=document.createElement('button');
        btn.type='button'; btn.className='danger'; btn.textContent=(typeof t==='function'?t('jobs_cancel'):'Cancel');
        btn.onclick=async()=>{
          if(!confirm((typeof t==='function'?t('jobs_cancel_confirm'):'Cancel job?')+' '+j.id)) return;
          try{ await jobCancel(j.id); await refreshJobsPanel(); }catch(e){ alert(e.message||e); }
        };
        row.appendChild(btn);
      }
      el.appendChild(row);
    });
  }catch(e){ el.innerHTML='<span class="note">'+escHtml(String(e))+'</span>'; }
}
function escHtml(s){ return String(s).replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c])); }

async function jobRun(type, params, onUpdate){
  const started=await jobStart(type, params);
  return jobWait(started.id, onUpdate);
}
function jobLogToUI(j){
  if(!j||!j.log) return;
  const lines=j.log.split('\n');
  const last=lines.slice(-8).join('\n');
  if(last) line(last);
}

async function streamPost(url, body, btn){
  /* All long deploys go through /v1/jobs only (no HTTP stream). Short sync allowlist: site. */
  if(btn) btn.disabled=true;
  const jobMap={
    '/v1/edge':'edge.provision',
    '/v1/edge/offline-prep':'edge.offline_prep',
    '/v1/fleet':'fleet.deploy',
    '/v1/primary':'primary.deploy',
    '/v1/secondary':'secondary.deploy',
    '/v1/build-host':'buildhost.deploy'
  };
  if(jobMap[url]){
    const typ=jobMap[url];
    const isEdge = typ==='edge.provision';
    if(isEdge) setEdgeRunStatus('run', t('edge_run_working'));
    const logSec=document.getElementById('deployLogSec');
    if(logSec){ logSec.classList.add('deploy-log-active'); try{ logSec.scrollIntoView({behavior:'smooth', block:'nearest'}); }catch(e){} }
    resetLog(); line('JOB '+typ+' (thin UI — refresh-safe)');
    try{ sessionStorage.setItem('nd_web_last_job_type', typ); }catch(e){}
    let failed=false;
    try{
      const params=Object.assign({}, body||{});
      if(params.router_password && !params.router_pass) params.router_pass=params.router_password;
      if(params.key_passphrase && !params.key_pass) params.key_pass=params.key_passphrase;
      const j=await jobRun(typ, params, j2=>{
        if(j2.log){ resetLog(); line(j2.log); }
        (j2.steps||[]).forEach(s=>chip(s.id, s.done?'ok':(s.message?'run':'')));
        (j2.steps||[]).forEach(s=>{ if(s.message) line('step '+s.id+(s.done?' done':'')+': '+s.message); });
      });
      failed = j.status!=='succeeded';
      if(j.error) line('ERROR '+j.error);
      if(j.log && !logEl.textContent.includes(j.log.slice(0,40))) line(j.log);
      line('status '+j.status+' id='+(j.id||''));
    }catch(e){ line('error: '+e); failed=true; }
    if(btn) btn.disabled=false;
    if(logSec) logSec.classList.remove('deploy-log-active');
    if(isEdge) setEdgeRunStatus(failed?'err':'ok', t(failed?'edge_run_err':'edge_run_done'));
    try{refreshJobsPanel();}catch(e){}
    return failed?'':('job '+typ+' ok');
  }
  /* Short sync only (no job type yet) */
  const syncOK={ '/v1/site':1 };
  if(!syncOK[url]){
    line('error: no job mapping for '+url+' — refused stream fallback');
    if(btn) btn.disabled=false;
    return '';
  }
  const logSec=document.getElementById('deployLogSec');
  if(logSec){ logSec.classList.add('deploy-log-active'); }
  resetLog(); line('POST '+url+' (sync short)');
  let failed=false;
  try{
    if(!ND_TOKEN){ line('unauthorized: empty operator token — hard-refresh'); failed=true; }
    else {
      const res=await fetch(url,{method:'POST',headers:{'Content-Type':'application/json','X-Netductor-Token':ND_TOKEN},body:JSON.stringify(body||{})});
      const text=await res.text();
      line(text||('HTTP '+res.status));
      if(!res.ok) failed=true;
    }
  }catch(e){ line('error: '+e); failed=true; }
  if(btn) btn.disabled=false;
  if(logSec) logSec.classList.remove('deploy-log-active');
  try{refreshJobsPanel();}catch(e){}
  return failed?'':'ok';
}

function saveForm(id,fd){ const o={}; for(const [k,v] of fd.entries()){ if(/password|passphrase|token|session/.test(k))continue; o[k]=v;} localStorage.setItem('nd_op_'+id,JSON.stringify(o)); }
function loadForm(id,form){ try{ const o=JSON.parse(localStorage.getItem('nd_op_'+id)||'{}'); Object.keys(o).forEach(k=>{ const el=form.elements.namedItem(k); if(!el)return; if(el.type==='checkbox')el.checked=true; else el.value=o[k]; }); }catch(e){} }

function tgFieldsOk(fd){
  if(fd.get('with_telegram')!=='on') return true;
  const tok=(fd.get('tg_token')||'').trim(), adm=(fd.get('tg_admin')||'').trim();
  if(!tok||!adm){ alert('Telegram: bot token and admin user id required'); return false; }
  return true;
}
function bindTgToggle(cbId, fieldsId, noteId){
  const cb=document.getElementById(cbId), box=document.getElementById(fieldsId), note=noteId?document.getElementById(noteId):null;
  if(!cb||!box) return;
  const sync=()=>{ box.hidden=!cb.checked; if(note) note.hidden=!cb.checked; };
  cb.addEventListener('change', sync); sync();
}
bindTgToggle('fleet_with_tg','fleet_tg_fields','fleet_tg_note');
bindTgToggle('pri_with_tg','pri_tg_fields',null);

document.getElementById('form-fleet').onsubmit=async e=>{ e.preventDefault(); const fd=new FormData(e.target); if(!tgFieldsOk(fd)) return; saveForm('fleet',fd);
  await streamPost('/v1/fleet',{ do_primary:fd.get('do_primary')==='on', do_secondary:fd.get('do_secondary')==='on', do_build:fd.get('do_build')==='on',
    build_host:fd.get('build_host'), build_password:fd.get('build_password'), build_label:fd.get('build_label')||'build-1',
    primary_host:fd.get('primary_host'), primary_password:fd.get('primary_password'),
    secondary_host:fd.get('secondary_host'), secondary_password:fd.get('secondary_password'),
    domain_base:fd.get('domain_base'), domain_primary:fd.get('domain_primary'), domain_vpn:fd.get('domain_vpn'), domain_redirect:fd.get('domain_redirect'), le_email:fd.get('le_email'), sni:fd.get('sni'), ssh_port:fd.get('ssh_port'), redirect_https_port:fd.get('redirect_https_port'), agent_mtls_port:fd.get('agent_mtls_port'), lampac_port:fd.get('lampac_port'), key:fd.get('key'), key_passphrase:fd.get('key_passphrase'),
    cf_proxy:fd.get('cf_proxy')==='on',
    with_lampac:fd.get('with_lampac')==='on', with_git:fd.get('with_git')==='on',
    with_telegram:fd.get('with_telegram')==='on',
    tg_token:(fd.get('tg_token')||'').trim(), tg_admin:(fd.get('tg_admin')||'').trim() }, e.submitter); };
document.getElementById('form-primary').onsubmit=async e=>{ e.preventDefault(); const fd=new FormData(e.target); if(!tgFieldsOk(fd)) return; saveForm('primary',fd);
  await streamPost('/v1/primary',{ host:fd.get('host'), password:fd.get('password'), domain_base:fd.get('domain_base'), domain_primary:fd.get('domain_primary'), domain_vpn:fd.get('domain_vpn'), domain_redirect:fd.get('domain_redirect'), le_email:fd.get('le_email'),
    sni:fd.get('sni'), ssh_port:fd.get('ssh_port'), redirect_https_port:fd.get('redirect_https_port'), agent_mtls_port:fd.get('agent_mtls_port'), lampac_port:fd.get('lampac_port'), key:fd.get('key'),
    cf_proxy:fd.get('cf_proxy')==='on',
    with_lampac:fd.get('with_lampac')==='on', with_git:fd.get('with_git')==='on',
    with_telegram:fd.get('with_telegram')==='on',
    tg_token:(fd.get('tg_token')||'').trim(), tg_admin:(fd.get('tg_admin')||'').trim() }, e.submitter); };

document.getElementById('form-build')&&(document.getElementById('form-build').onsubmit=async e=>{ e.preventDefault(); const fd=new FormData(e.target); saveForm('build',fd);
  await streamPost('/v1/build-host',{ host:fd.get('host'), password:fd.get('password'), user:fd.get('user')||'root', port:parseInt(fd.get('port')||'22',10),
    primary_host:fd.get('primary_host'), primary_key:fd.get('primary_key'), label:fd.get('label')||'' });
});

document.getElementById('form-secondary').onsubmit=async e=>{ e.preventDefault(); const fd=new FormData(e.target); saveForm('secondary',fd);
  await streamPost('/v1/secondary',{ primary_host:fd.get('primary_host'), primary_key:fd.get('primary_key'), secondary_host:fd.get('secondary_host'), secondary_password:fd.get('secondary_password') }, e.submitter); };
document.getElementById('form-site').onsubmit=async e=>{ e.preventDefault(); const fd=new FormData(e.target); saveForm('site',fd);
  await streamPost('/v1/site',{
    site_id:fd.get('site_id'), name:fd.get('name'), rpi_id:fd.get('rpi_id'), mikrotik_id:fd.get('mikrotik_id'),
    rpi_lan:fd.get('rpi_lan'), mt_host:fd.get('mt_host'), mt_user:fd.get('mt_user'), mt_password:fd.get('mt_password'),
    mt_port:fd.get('mt_port'), do_push:fd.get('do_push')==='on', operator_key:fd.get('operator_key')
  }, e.submitter); };
document.getElementById('form-mt').onsubmit=async e=>{ e.preventDefault(); const fd=new FormData(e.target);
  const res=await fetch('/v1/mikrotik',{method:'POST',headers:{'Content-Type':'application/json','X-Netductor-Token':ND_TOKEN},
    body:JSON.stringify({host:fd.get('host'),user:fd.get('user'),password:fd.get('password'),port:fd.get('port'),action:fd.get('action')})});
  const j=await res.json(); document.getElementById('mtResult').textContent=j.output||j.error||JSON.stringify(j); };
function fleetSettings(){
  let fleet={}, settings={};
  try{ fleet=JSON.parse(localStorage.getItem('nd_op_fleet')||'{}'); }catch(e){}
  try{ settings=JSON.parse(localStorage.getItem('nd_op_settings')||'{}'); }catch(e){}
  return {fleet, settings};
}
function defaultPrimaryKey(fleet, settings){
  return fleet.primary_key || fleet.key || settings.primary_key || settings.key
    || '~/.ssh/netductor_primary';
}

async function loadNetmap(){
  const box=document.getElementById('netmapBox');
  if(!box) return;
  try{
    const r=await fetch('/v1/netmap?lang='+(uiLang==='ru'?'ru':'en'),{headers:{'X-Netductor-Token':ND_TOKEN}});
    if(!r.ok) throw new Error(String(r.status));
    const j=await r.json();
    if(j.html) box.innerHTML=j.html;
    else if(j.rows){
      box.innerHTML='<table class="netmap"><tbody>'+j.rows.map(row=>
        '<tr><td>'+row.network+'</td><td><code>'+row.cidr+'</code></td><td>'+row.purpose+'</td></tr>'
      ).join('')+'</tbody></table>';
    }
  }catch(e){
    box.innerHTML='<p class="note">netmap: '+String(e)+'</p>';
  }
}

async function prefillAllFromConfig(){
  const {fleet, settings}=fleetSettings();
  let cfg={};
  try{
    const r=await fetch('/v1/config',{headers:{'X-Netductor-Token':ND_TOKEN}});
    if(r.ok) cfg=await r.json();
  }catch(e){}
  const host = (fleet.primary_host || settings.primary_host || cfg.remote_host || '').trim();
  const key = (defaultPrimaryKey(fleet, settings) || cfg.remote_key || '').trim();
  const user = (settings.user || cfg.remote_user || 'root').trim();
  const sec = (fleet.secondary_host || settings.secondary_host || cfg.secondary_host || '').trim();
  const edgeId = (cfg.last_edge_id || '').trim();

  function setEmpty(form, name, val){
    if(!form || val==null || val==='') return;
    const el=form.elements.namedItem(name);
    if(!el) return;
    if(el.type==='checkbox') return;
    if(String(el.value||'').trim()==='') el.value=val;
  }
  function setEmptyId(id, val){
    if(val==null || val==='') return;
    const el=document.getElementById(id);
    if(el && String(el.value||'').trim()==='') el.value=val;
  }

  // Settings tab
  setEmptyId('set_primary_host', host);
  setEmptyId('set_key', key);
  setEmptyId('set_user', user);
  setEmptyId('set_secondary_host', sec);

  // Fleet / Primary / Secondary / Build / Edge / LuCI / Creds
  const maps = [
    ['form-fleet', {primary_host:host, secondary_host:sec, key:key}],
    ['form-primary', {host:host, key:key}],
    ['form-secondary', {primary_host:host, primary_key:key, secondary_host:sec}],
    ['form-build', {primary_host:host, primary_key:key}],
    ['form-edge', {primary_host:host, primary_key:key, device_id:edgeId}],
    ['form-edge-luci', {primary_key:key, device_id:edgeId}],
    ['form-creds', {host:host, key:key}],
  ];
  for(const [fid, fields] of maps){
    const form=document.getElementById(fid);
    if(!form) continue;
    for(const [n,v] of Object.entries(fields)) setEmpty(form, n, v);
  }
  // edge server_url from primary host
  const form=document.getElementById('form-edge');
  if(form){
    const su=form.elements.namedItem('server_url');
    const ph=form.elements.namedItem('primary_host');
    if(su && !su.value && ph && ph.value){
      const h=(ph.value||'').trim().replace(/^https?:\/\//,'').split('/')[0].split(':')[0];
      if(h) su.value='https://'+h+':8789';
    }
  }
  // lang from file if no explicit localStorage choice yet and lang is ru|en
  if(!localStorage.getItem('nd_op_lang') && (cfg.lang==='ru'||cfg.lang==='en')){
    uiLang=cfg.lang;
    try{ applyI18n(); }catch(e){}
  }
  // mirror into local settings for tunnel helpers
  if(host || key){
    const s=settings||{};
    if(host && !s.primary_host) s.primary_host=host;
    if(key && !s.key) s.key=key;
    if(user && !s.user) s.user=user;
    if(sec && !s.secondary_host) s.secondary_host=sec;
    try{ saveSettingsObj(s); }catch(e){}
  }
}

async function saveConfigToDisk(extra){
  const s=settings();
  const body=Object.assign({
    remote_host: s.primary_host||'',
    remote_key: s.key||'',
    remote_user: s.user||'root',
    secondary_host: s.secondary_host||'',
    lang: uiLang||'auto'
  }, extra||{});
  // also from settings form if present
  const fh=document.getElementById('set_primary_host');
  const fk=document.getElementById('set_key');
  const fu=document.getElementById('set_user');
  const fs=document.getElementById('set_secondary_host');
  if(fh&&fh.value) body.remote_host=fh.value.trim();
  if(fk&&fk.value) body.remote_key=fk.value.trim();
  if(fu&&fu.value) body.remote_user=fu.value.trim();
  if(fs&&fs.value) body.secondary_host=fs.value.trim();
  const res=await fetch('/v1/config',{method:'PUT',headers:{'Content-Type':'application/json','X-Netductor-Token':ND_TOKEN},body:JSON.stringify(body)});
  const j=await res.json().catch(()=>({}));
  if(!res.ok) throw new Error(j.error||('HTTP '+res.status));
  return j;
}

function syncEdgePanels(){ if(!document.getElementById('edge_net_fields')) return; // legacy
  const net=document.getElementById('edge_net_cfg');
  const g=document.getElementById('edge_guest');
  const nf=document.getElementById('edge_net_fields');
  const gf=document.getElementById('edge_guest_fields');
  if(nf) nf.hidden = !(net && net.checked);
  if(gf) gf.hidden = !(g && g.checked);
  const sk=document.getElementById('edge_skip_root');
  const nr=document.getElementById('edge_new_root');
  const nri=document.getElementById('edge_new_root_input');
  if(nr) nr.hidden = !!(sk && sk.checked);
  if(nri){ nri.required = !(sk && sk.checked); if(sk&&sk.checked) nri.value=''; }
  const wp=document.getElementById('edge_wan_proto');
  const st=document.getElementById('edge_wan_static');
  const pp=document.getElementById('edge_wan_pppoe');
  const v=wp ? wp.value : 'dhcp';
  if(st) st.hidden = v!=='static';
  if(pp) pp.hidden = v!=='pppoe';
}
['edge_net_cfg','edge_guest','edge_wan_proto','edge_skip_root'].forEach(id=>{
  const el=document.getElementById(id);
  if(el) el.addEventListener('change', syncEdgePanels);
});
document.getElementById('form-edge').onsubmit=async e=>{ e.preventDefault(); const fd=new FormData(e.target); saveForm('edge',fd);
  await streamPost('/v1/edge',{
    offline:!!document.getElementById('edge_offline')?.checked,
    router_host:fd.get('router_host'), router_password:fd.get('router_password')||'',
    new_root_password:(fd.get('skip_root_pass')==='on'?'':(fd.get('new_root_password')||'')),
    skip_root_pass:fd.get('skip_root_pass')==='on',
    guest_hidden:fd.get('guest_hidden')||'0',
    device_id:fd.get('device_id'), agent_arch:fd.get('agent_arch')||'auto',
    primary_host:fd.get('primary_host'), primary_key:fd.get('primary_key'),
    server_url:fd.get('server_url'), key_passphrase:fd.get('key_passphrase'),
    net_configure:fd.get('net_configure')==='on', guest_enable:fd.get('guest_enable')==='on',
    reboot:fd.get('reboot')==='on',
    lan_ip:fd.get('lan_ip'), lan_mask:fd.get('lan_mask'),
    dhcp_start:fd.get('dhcp_start'), dhcp_limit:fd.get('dhcp_limit'),
    wifi_ssid_24:fd.get('wifi_ssid_24'), wifi_key_24:fd.get('wifi_key_24'),
    wifi_ssid_5:fd.get('wifi_ssid_5'), wifi_key_5:fd.get('wifi_key_5'),
    wan_proto:fd.get('wan_proto')||'dhcp',
    wan_ip:fd.get('wan_ip'), wan_mask:fd.get('wan_mask'),
    wan_gateway:fd.get('wan_gateway'), wan_dns:fd.get('wan_dns'),
    pppoe_user:fd.get('pppoe_user'), pppoe_pass:fd.get('pppoe_pass'),
    guest_ssid:fd.get('guest_ssid'), guest_pin:fd.get('guest_pin'), guest_psk:fd.get('guest_psk')
  }, e.submitter); };

async function instNvrRefreshTable(){
  const j=await nodeFetch('/api/nvr/cameras');
  const box=document.getElementById('instNvrTable');
  if(!box) return j;
  const cams=(j&&j.cameras)||[];
  if(!cams.length){ box.innerHTML='<p class="muted">No cameras</p>'; return j; }
  let h='<table class="striped"><tr><th>id</th><th>name</th><th>ip</th><th>rec</th><th></th></tr>';
  cams.forEach(c=>{
    h+=`<tr><td><code>${c.id||''}</code></td><td>${c.name||''}</td><td>${c.lan_ip||''}</td><td>${c.record?'🟢':'—'}</td>`+
      `<td><button type="button" data-cam="${c.id}" class="inst-nvr-pick">Select</button></td></tr>`;
  });
  h+='</table>';
  box.innerHTML=h;
  box.querySelectorAll('.inst-nvr-pick').forEach(btn=>{
    btn.onclick=()=>{ const el=document.getElementById('instNvrCamId'); if(el) el.value=btn.getAttribute('data-cam'); };
  });
  return j;
}
document.getElementById('form-nvr')&&(document.getElementById('form-nvr').onsubmit=async e=>{
  e.preventDefault(); const fd=new FormData(e.target); saveForm('nvr',fd);
  const btn=e.submitter; if(btn) btn.disabled=true;
  try{
    const res=await nodeFetch('/api/nvr/cameras',{method:'POST',body:JSON.stringify({
      name:(fd.get('cam_name')||'').trim(), site_id:(fd.get('site_id')||fd.get('device_id')||'').trim(),
      lan_ip:(fd.get('cam_ip')||'').trim(), rtsp_user:(fd.get('rtsp_user')||'').trim(),
      rtsp_password:(fd.get('cam_pass')||'').trim(), cloud_password:(fd.get('cloud_pass')||'').trim(),
      rtsp_path:(fd.get('rtsp_path')||'/stream1').trim(), enabled:true, record:true
    })});
    showControl(res);
    await instNvrRefreshTable();
  }catch(err){ showControl({error:String(err)}); }
  if(btn) btn.disabled=false;
});
document.getElementById('instNvrRefresh')?.addEventListener('click', async()=>{
  try{ showControl(await instNvrRefreshTable()); }catch(e){ showControl({error:String(e)}); }
});
document.getElementById('instWizLeases')?.addEventListener('click', async()=>{
  const did=(document.getElementById('instWizDid')?.value||document.getElementById('instNvrDid')?.value||'').trim();
  const box=document.getElementById('instWizList');
  if(!did){ showControl({error:'device_id'}); return; }
  if(box) box.textContent='waiting for dhcp_leases…';
  try{
    const j=await nodeFetch('/api/nvr/site/leases',{method:'POST',body:JSON.stringify({device_id:did,wait:true,timeout_sec:90})});
    const leases=(j&&j.leases)||[];
    if(!box){ showControl(j); return; }
    if(!leases.length){ box.textContent=JSON.stringify(j,null,2); return; }
    let h='<table class="striped"><tr><th>host</th><th>ip</th><th>mac</th><th></th></tr>';
    leases.forEach((L,i)=>{
      h+=`<tr><td>${L.hostname||'—'}</td><td>${L.ip||''}</td><td><code>${L.mac||''}</code></td>`+
        `<td><button type="button" data-i="${i}" class="inst-lease-use">Use</button></td></tr>`;
    });
    h+='</table>';
    box.innerHTML=h;
    box.querySelectorAll('.inst-lease-use').forEach(btn=>{
      btn.onclick=()=>{
        const L=leases[+btn.getAttribute('data-i')];
        if(!L) return;
        const set=(id,v)=>{ const el=document.getElementById(id); if(el) el.value=v||''; };
        set('instWizIp', L.ip); set('instWizMac', L.mac);
        set('instWizName', (L.hostname&&L.hostname!=='*')?L.hostname:('cam-'+(L.mac||'').replace(/:/g,'').slice(-6)));
        set('instNvrIp', L.ip); set('instNvrName', document.getElementById('instWizName').value);
      };
    });
    showControl({ok:true, count:leases.length, cmd_id:j.cmd_id});
  }catch(e){ showControl({error:String(e)}); }
});
document.getElementById('instWizAdd')?.addEventListener('click', async()=>{
  try{
    const res=await nodeFetch('/api/nvr/cameras',{method:'POST',body:JSON.stringify({
      name:(document.getElementById('instWizName')?.value||'').trim(),
      lan_ip:(document.getElementById('instWizIp')?.value||'').trim(),
      mac:(document.getElementById('instWizMac')?.value||'').trim(),
      rtsp_password:(document.getElementById('instWizPass')?.value||'').trim(),
      site_id:(document.getElementById('instWizDid')?.value||document.getElementById('instNvrDid')?.value||'').trim(),
      enabled:true, record:true, rtsp_path:'/stream1'
    })});
    showControl(res); await instNvrRefreshTable();
  }catch(e){ showControl({error:String(e)}); }
});
document.getElementById('instNvrPlay')?.addEventListener('click', ()=>{
  const id=(document.getElementById('instNvrCamId')?.value||'').trim();
  const v=document.getElementById('instNvrVideo');
  if(!id||!v) return;
  v.src='/api/nvr/stream?id='+encodeURIComponent(id);
  v.play().catch(()=>{});
});
document.getElementById('instNvrRec')?.addEventListener('click', async()=>{
  const id=(document.getElementById('instNvrCamId')?.value||'').trim();
  try{ showControl(await nodeFetch('/api/nvr/recorder/start',{method:'POST',body:JSON.stringify({id})})); }catch(e){ showControl({error:String(e)}); }
});
document.getElementById('instNvrStop')?.addEventListener('click', async()=>{
  const id=(document.getElementById('instNvrCamId')?.value||'').trim();
  try{ showControl(await nodeFetch('/api/nvr/recorder/stop',{method:'POST',body:JSON.stringify({id})})); }catch(e){ showControl({error:String(e)}); }
});

document.getElementById('form-creds').onsubmit=async e=>{ e.preventDefault(); const fd=new FormData(e.target); saveForm('creds',fd); const btn=e.submitter; btn.disabled=true;
  try{ const res=await fetch('/v1/credentials',{method:'POST',headers:{'Content-Type':'application/json','X-Netductor-Token':ND_TOKEN},body:JSON.stringify({role:fd.get('role'),host:fd.get('host'),key:fd.get('key'),key_passphrase:fd.get('key_passphrase')})});
    const j=await res.json(); document.getElementById('credResult').textContent=j.path?('OK → '+j.path):(j.error||JSON.stringify(j)); }catch(err){ document.getElementById('credResult').textContent=String(err); }
  btn.disabled=false; };

function loadSettingsForm(){
  const s=settings();
  const map={set_primary_host:'primary_host',set_key:'key',set_user:'user',set_api_port:'api_port',set_api_base:'api_base',set_secondary_host:'secondary_host',set_vpn_host:'vpn_host',set_node_session:'node_session'};
  Object.keys(map).forEach(id=>{ const el=document.getElementById(id); if(!el)return; el.value=s[map[id]]||(id==='set_key'?'~/.ssh/netductor_primary':id==='set_user'?'root':id==='set_api_port'?'8787':id==='set_api_base'?'http://127.0.0.1:8787':''); });
  const pv=document.getElementById('set_prefer_vpn'); if(pv) pv.value=(s.prefer_vpn==='0'?'0':'1');
  /* ui skins removed */
}
(function(){ const el=document.getElementById('set_show_advanced'); if(el){ el.checked=!!(settings().show_advanced===true||settings().show_advanced==='1'); } })();
document.getElementById('form-settings').onsubmit=async e=>{
  e.preventDefault(); const fd=new FormData(e.target);
  saveSettingsObj({ show_advanced: fd.get('show_advanced')==='on', primary_host:fd.get('primary_host'), key:fd.get('key'), user:fd.get('user'),
    api_port:fd.get('api_port'), api_base:fd.get('api_base'), secondary_host:fd.get('secondary_host'),
    vpn_host:fd.get('vpn_host'), prefer_vpn:fd.get('prefer_vpn'), node_session:fd.get('node_session'), });
    const el=document.getElementById('settingsSaved');
  try{
    const j=await saveConfigToDisk({
      remote_host: fd.get('primary_host')||'',
      remote_key: fd.get('key')||'',
      remote_user: fd.get('user')||'root',
      secondary_host: fd.get('secondary_host')||'',
      lang: uiLang||'auto'
    });
    el.textContent='OK → '+(j.path||'operator.conf');
  }catch(err){
    el.textContent='local OK; disk: '+String(err);
  }
  updateTunnelHint();
  applyAdvancedVisibility();
};
function updateTunnelHint(){
  const s=settings();
  document.getElementById('tunnelCmd').textContent='host='+(s.primary_host||'—')+' vpn='+(s.vpn_host||'—')+' key='+(s.key||'~/.ssh/netductor_primary');
}
async function refreshTunnelBadge(){
  try{
    const res=await fetch('/v1/tunnel/status',{headers:{'X-Netductor-Token':ND_TOKEN}});
    const st=await res.json();
    const b=document.getElementById('tunnelBadge');
    if(st.port_open){ b.textContent='up :'+st.local_port+(st.via?(' ·'+st.via):''); b.className='chip ok'; }
    else if(st.process_up){ b.textContent='starting…'; b.className='chip run'; }
    else { b.textContent='down'; b.className='chip err'; }
    return st;
  }catch(e){ return null; }
}
async function ensureTunnel(){
  const s=settings();
  if(!s.primary_host && !s.vpn_host) return false;
  const res=await fetch('/v1/tunnel/start',{method:'POST',headers:{'Content-Type':'application/json','X-Netductor-Token':ND_TOKEN},
    body:JSON.stringify({ host:s.primary_host, vpn_host:s.vpn_host, prefer_vpn:s.prefer_vpn!=='0',
      user:s.user||'root', key:s.key, local_port:s.api_port||'8787', api_base:apiBase() })});
  await refreshTunnelBadge();
  return res.json();
}
async function nodeFetch(path, opts){
  opts=opts||{}; const s=settings();
  const headers={'X-Netductor-Token':ND_TOKEN,'X-Node-API-Base':apiBase()};
  if(s.node_session) headers['X-Node-Session']=s.node_session;
  if(opts.body) headers['Content-Type']='application/json';
  const res=await fetch('/v1/node'+path,{method:opts.method||'GET',headers,body:opts.body});
  const text=await res.text(); let data; try{data=JSON.parse(text)}catch(e){data=text}
  return {ok:res.ok,status:res.status,data};
}

let lastRaw=null;
function esc(s){ return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;'); }
function unwrapPayload(payload){
  // nodeFetch returns {ok,status,data}; prefer data for tables/doctor when object
  if(payload && typeof payload==='object' && !Array.isArray(payload) && payload.data!==undefined
    && (payload.ok!==undefined || payload.status!==undefined)){
    return payload.data;
  }
  return payload;
}
function formatPreviewText(payload){
  const u=unwrapPayload(payload);
  if(payload && payload.ok===false){
    const err=(u&&u.error)||payload.error||('HTTP '+(payload.status||''));
    if(payload.status===401) return (uiLang==='ru'
      ? '401 unauthorized — выдайте session (Operate → Issue session) и поднимите tunnel'
      : '401 unauthorized — Issue session + start tunnel first');
    return 'error: '+String(err).slice(0,200);
  }
  if(u && u.checks && u.summary){
    const s=u.summary;
    return 'Doctor · ok='+(s.ok||0)+' warn='+(s.warn||0)+' fail='+(s.fail||0);
  }
  if(typeof u==='string') return u.slice(0,300);
  if(u==null) return '—';
  try{
    if(typeof u==='object'){
      const keys=Object.keys(u).slice(0,8);
      const lines=keys.map(k=>{
        let v=u[k];
        if(v!==null && typeof v==='object') v=Array.isArray(v)?('['+v.length+']'):'{…}';
        return k+': '+v;
      });
      return lines.join('\n').slice(0,400);
    }
  }catch(e){}
  try{ return JSON.stringify(u).slice(0,300); }catch(e){ return String(u); }
}
function renderTables(payload){
  const box=document.getElementById('controlTables');
  const raw=document.getElementById('controlStatus');
  const btn=document.getElementById('btnShowRaw');
  lastRaw=payload; if(box) box.innerHTML=''; if(raw){ raw.classList.add('hidden'); raw.style.display='none'; } if(btn) btn.style.display='inline-block';
  // 401 / envelope errors — clear message instead of empty tables
  if(payload && typeof payload==='object' && payload.ok===false){
    const msg=formatPreviewText(payload);
    if(box) box.innerHTML='<div class="card" style="padding:.65rem;border-color:var(--err)"><b>'+esc(msg)+'</b></div>';
    if(raw){ raw.classList.remove('hidden'); raw.style.display='block'; raw.textContent=typeof payload==='string'?payload:JSON.stringify(payload,null,2); }
    return;
  }
  const data = unwrapPayload(payload);
  const tables=[];
  function walk(obj, path){
    if(Array.isArray(obj)){
      if(obj.length && typeof obj[0]==='object' && obj[0]!==null && !Array.isArray(obj[0])) tables.push({title:path||'items', rows:obj});
      else if(obj.length) tables.push({title:path||'list', rows:obj.map((v,i)=>({index:i,value:typeof v==='object'?JSON.stringify(v):v}))});
      return;
    }
    if(obj && typeof obj==='object'){
      const scalars={};
      for(const [k,v] of Object.entries(obj)){
        if(v!==null && typeof v==='object') walk(v, path?path+'.'+k:k);
        else scalars[k]=v;
      }
      if(Object.keys(scalars).length) tables.push({title:path||'summary', rows:[scalars]});
    }
  }
  walk(data,'');
  if(!tables.length){ if(raw){ raw.classList.remove('hidden'); raw.style.display='block'; raw.textContent=typeof payload==='string'?payload:JSON.stringify(payload,null,2);} if(btn) btn.style.display='none'; return; }
  for(const t of tables){
    const keys=[]; t.rows.forEach(r=>Object.keys(r||{}).forEach(k=>{ if(!keys.includes(k)) keys.push(k); }));
    const cols=keys.slice(0,12);
    const hasName=cols.includes('Name')||cols.includes('name');
    const hasDid=cols.includes('device_id')||cols.includes('ID')||cols.includes('id');
    if(hasName||hasDid) cols.push('_act');
    let h='<div class="card" style="padding:.75rem"><h3 style="margin:0 0 .5rem;font-size:.9rem">'+esc(t.title)+' <span class="note">('+t.rows.length+')</span></h3>';
    h+='<div style="overflow:auto"><table style="width:100%;border-collapse:collapse;font-size:.78rem"><thead><tr>'+cols.map(c=>'<th style="text-align:left;border-bottom:1px solid var(--border);padding:.35rem;color:var(--muted)">'+esc(c==='_act'?'':c)+'</th>').join('')+'</tr></thead><tbody>';
    t.rows.slice(0,200).forEach(r=>{
      h+='<tr>'+cols.map(c=>{
        if(c==='_act'){
          let btns='';
          const nm=r.Name||r.name; const did=r.device_id||r.ID||r.id;
          if(nm){ btns+='<button type="button" class="primary rowact" data-kind="vpn-disable" data-name="'+esc(nm)+'" style="padding:.15rem .35rem;font-size:.7rem">off</button> ';
                  btns+='<button type="button" class="primary rowact" data-kind="vpn-enable" data-name="'+esc(nm)+'" style="padding:.15rem .35rem;font-size:.7rem">on</button>'; }
          if(did && /pending/i.test(String(t.title))){
            btns+=' <button type="button" class="primary rowact" data-kind="edge-approve" data-id="'+esc(did)+'" style="padding:.15rem .35rem;font-size:.7rem">ok</button>';
            btns+=' <button type="button" class="primary rowact" data-kind="edge-deny" data-id="'+esc(did)+'" style="padding:.15rem .35rem;font-size:.7rem">no</button>';
          }
          return '<td style="border-bottom:1px solid var(--border);padding:.35rem;white-space:nowrap">'+btns+'</td>';
        }
        let v=r[c]; if(v===null||v===undefined) v=''; else if(typeof v==='object') v=JSON.stringify(v);
        return '<td style="border-bottom:1px solid var(--border);padding:.35rem;vertical-align:top">'+esc(v)+'</td>';
      }).join('')+'</tr>';
    });
    h+='</tbody></table></div></div>'; if(box) box.innerHTML+=h;
  }
}

function formatDoctorWeb(x){
  x=unwrapPayload(x);
  if(!x||!x.checks) return null;
  const s=x.summary||{};
  let h='<div class="card"><b>Doctor</b> '+esc(x.role||'')+' @ '+esc(x.host||'')+
    ' · ok='+(s.ok||0)+' warn='+(s.warn||0)+' fail='+(s.fail||0)+'</div>';
  h+='<table class="grid"><tr><th>check</th><th>status</th></tr>';
  (x.checks||[]).forEach(c=>{
    if(c.status==='ok') return;
    h+='<tr><td>'+esc(c.id||'')+'</td><td>'+esc(c.status||'')+'</td></tr>';
  });
  h+='</table>';
  return h;
}
function esc(s){ return String(s).replace(/[&<>"]/g,c=>({ '&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;' }[c])); }

/* ===== Result dock (VS Code–style) ===== */
const RD_HIST_KEY='nd_op_result_hist';
const RD_LS_KEY='nd_op_result_ui';
let rdState={visible:true, position:'bottom', mode:'both', size:220};
let rdLastAct='';
let rdLastBtn=null;

function rdLoadLocal(){
  try{
    const o=JSON.parse(localStorage.getItem(RD_LS_KEY)||'{}');
    if(o.position) rdState.position=o.position;
    if(o.mode) rdState.mode=o.mode;
    if(o.size) rdState.size=+o.size||220;
    if(typeof o.visible==='boolean') rdState.visible=o.visible;
  }catch(e){}
}
function rdSaveLocal(){
  try{ localStorage.setItem(RD_LS_KEY, JSON.stringify(rdState)); }catch(e){}
}
async function rdPersistDisk(){
  rdSaveLocal();
  try{
    await fetch('/v1/config',{
      method:'PUT',
      headers:{'Content-Type':'application/json','X-Netductor-Token':ND_TOKEN},
      body:JSON.stringify({
        result_visible: rdState.visible?'1':'0',
        result_position: rdState.position,
        result_mode: rdState.mode,
        result_size: String(rdState.size||220),
      })
    });
  }catch(e){}
}
function rdHistLoad(){
  try{ return JSON.parse(localStorage.getItem(RD_HIST_KEY)||'[]'); }catch(e){ return []; }
}
function rdHistPush(act, payload){
  const list=rdHistLoad();
  const entry={ts:Date.now(), act:act||'—', summary: rdSummary(payload), raw: payload};
  list.unshift(entry);
  while(list.length>40) list.pop();
  try{ localStorage.setItem(RD_HIST_KEY, JSON.stringify(list)); }catch(e){}
  rdRenderHist();
}
function rdSummary(x){
  if(x==null) return '—';
  if(typeof x==='string') return x.slice(0,80);
  if(x.error) return 'error: '+String(x.error).slice(0,60);
  if(x.loading) return '… '+x.loading;
  if(x.checks&&x.summary) return 'doctor ok='+(x.summary.ok||0)+' fail='+(x.summary.fail||0);
  if(x.ok===true) return 'ok';
  try{ return JSON.stringify(x).slice(0,80); }catch(e){ return 'result'; }
}
function rdRenderHist(){
  const box=document.getElementById('rdHistList'); if(!box) return;
  const list=rdHistLoad();
  if(!list.length){ box.innerHTML='<span class="note">—</span>'; return; }
  box.innerHTML=list.map((e,i)=>{
    const when=new Date(e.ts).toLocaleTimeString();
    return '<button type="button" data-hist="'+i+'"><div class="t">'+esc(e.act)+'</div><div class="m">'+esc(when)+' · '+esc(e.summary)+'</div></button>';
  }).join('');
  box.querySelectorAll('[data-hist]').forEach(b=>{
    b.onclick=()=>{
      const e=rdHistLoad()[+b.dataset.hist];
      if(!e) return;
      rdLastAct=e.act;
      showControl(e.raw, {fromHist:true, act:e.act});
      document.getElementById('rdTabOut')?.click();
    };
  });
}
function rdApplyChrome(){
  const dock=document.getElementById('resultDock'); if(!dock) return;
  dock.classList.remove('pos-bottom','pos-right','pos-top','hidden','compact-tools');
  if(!rdState.visible || rdState.mode==='inline'){
    if(rdState.mode==='inline') dock.classList.add('hidden');
    else if(!rdState.visible) dock.classList.add('hidden');
  }
  if(rdState.mode!=='inline' && rdState.visible){
    dock.classList.add('pos-'+(rdState.position||'bottom'));
  } else if(rdState.mode!=='inline' && !rdState.visible){
    dock.classList.add('hidden');
  }
  // right dock: hide wide toolbar, only ⋮ menu
  if((rdState.position||'bottom')==='right') dock.classList.add('compact-tools');
  const size=Math.max(120, Math.min(rdState.position==='right'?640:480, +rdState.size||220));
  dock.style.setProperty('--rd-size', size+'px');
  const pos=document.getElementById('rdPos'); if(pos) pos.value=rdState.position||'bottom';
  const mode=document.getElementById('rdMode'); if(mode) mode.value=rdState.mode||'both';
  const hdr=document.getElementById('btnHeaderResult');
  if(hdr) hdr.classList.toggle('on', rdState.visible && rdState.mode!=='inline');
  const sb=document.getElementById('sbResult');
  if(sb) sb.classList.toggle('on', rdState.visible && rdState.mode!=='inline');
}
function rdShow(force){
  if(rdState.mode==='inline' && !force) return;
  rdState.visible=true;
  rdApplyChrome();
  rdPersistDisk();
}
function rdHide(){
  rdState.visible=false;
  rdApplyChrome();
  rdPersistDisk();
}
function rdToggle(){
  if(rdState.mode==='inline'){
    rdState.mode='both';
    rdState.visible=true;
  } else {
    rdState.visible=!rdState.visible;
  }
  rdApplyChrome();
  rdPersistDisk();
}
function rdSetBusy(btn, act, on){
  document.querySelectorAll('button.op-link.busy,button.primary.busy').forEach(b=>b.classList.remove('busy'));
  if(btn && on){ btn.classList.add('busy'); rdLastBtn=btn; }
  const dot=document.getElementById('rdDot');
  if(dot){ dot.className='rd-dot'+(on?' run':''); }
  const lab=document.getElementById('rdActLabel');
  if(lab) lab.textContent=act?('· '+act):'';
}
function rdFlash(btn, ok){
  if(!btn) return;
  btn.classList.remove('busy');
  btn.classList.add(ok?'ok-flash':'err-flash');
  setTimeout(()=>btn.classList.remove('ok-flash','err-flash'), 1600);
}
function rdClearInlinePreviews(){
  document.querySelectorAll('.inline-res').forEach(el=>el.remove());
}
function rdInlinePreview(btn, text){
  if(rdState.mode!=='inline' && rdState.mode!=='both') return;
  if(!btn) return;
  rdClearInlinePreviews();
  const box=document.createElement('div');
  box.className='inline-res';
  // always under the clicked control, not at top of card
  btn.insertAdjacentElement('afterend', box);
  box.innerHTML='<div class="h"><span>'+esc(rdLastAct||'…')+'</span><a href="#" class="rd-open-full" style="color:#a8c0d8;text-decoration:none">'+(uiLang==='ru'?'открыть':'open')+' →</a></div><pre style="margin:0;font-size:.72rem;white-space:pre-wrap">'+esc(text).slice(0,400)+'</pre>';
  box.querySelector('.rd-open-full')?.addEventListener('click',(ev)=>{
    ev.preventDefault();
    if(rdState.mode==='inline'){ rdState.mode='both'; }
    rdShow(true);
  });
}
function showControl(x, opts){
  opts=opts||{};
  const act=opts.act||rdLastAct||'';
  const box=document.getElementById('controlTables');
  if(x && x.loading){
    rdSetBusy(rdLastBtn, x.loading, true);
    if(rdState.mode!=='inline') rdShow();
    if(box) box.innerHTML='<span class="note">… '+esc(x.loading)+'</span>';
    const dot=document.getElementById('rdDot'); if(dot) dot.className='rd-dot run';
    return;
  }
  const isErr=!!(x && (x.error || x.ok===false));
  rdSetBusy(rdLastBtn, act, false);
  rdFlash(rdLastBtn, !isErr);
  const dot=document.getElementById('rdDot'); if(dot) dot.className='rd-dot '+(isErr?'err':'ok');
  if(rdState.mode!=='inline') rdShow();
  const unwrapped=unwrapPayload(x);
  if(box && unwrapped && unwrapped.checks){ box.innerHTML=formatDoctorWeb(x)||''; }
  renderTables(x);
  lastRaw=x;
  if(!opts.fromHist) rdHistPush(act, x);
  rdInlinePreview(rdLastBtn, formatPreviewText(x));
  const lab=document.getElementById('rdActLabel'); if(lab) lab.textContent=act?('· '+act):'';
}

function initResultDock(){
  rdLoadLocal();
  // merge from disk config when available
  (async()=>{
    try{
      const res=await fetch('/v1/config',{headers:{'X-Netductor-Token':ND_TOKEN}});
      const j=await res.json();
      if(j.result_position) rdState.position=j.result_position;
      if(j.result_mode) rdState.mode=j.result_mode;
      if(j.result_size) rdState.size=+j.result_size||rdState.size;
      if(j.result_visible==='0') rdState.visible=false;
      if(j.result_visible==='1') rdState.visible=true;
      rdApplyChrome();
    }catch(e){ rdApplyChrome(); }
  })();
  rdApplyChrome();
  rdRenderHist();
  document.getElementById('btnHeaderResult')?.addEventListener('click', rdToggle);
  document.getElementById('sbResult')?.addEventListener('click', rdToggle);
  document.getElementById('rdClose')?.addEventListener('click', rdHide);

  const menuBtn=document.getElementById('rdMenuBtn');
  const menu=document.getElementById('rdMenu');
  menuBtn?.addEventListener('click',(e)=>{ e.stopPropagation(); menu?.classList.toggle('hidden'); });
  document.addEventListener('click',()=>menu?.classList.add('hidden'));
  menu?.addEventListener('click',e=>e.stopPropagation());
  const syncPos=(v)=>{ rdState.position=v; rdApplyChrome(); rdPersistDisk();
    const a=document.getElementById('rdPos'); const b=document.getElementById('rdPosM'); if(a)a.value=v; if(b)b.value=v; };
  const syncMode=(v)=>{ rdState.mode=v; if(v!=='inline') rdState.visible=true; rdApplyChrome(); rdPersistDisk();
    const a=document.getElementById('rdMode'); const b=document.getElementById('rdModeM'); if(a)a.value=v; if(b)b.value=v; };
  document.getElementById('rdPosM')?.addEventListener('change',e=>syncPos(e.target.value));
  document.getElementById('rdModeM')?.addEventListener('change',e=>syncMode(e.target.value));
  document.getElementById('rdRawM')?.addEventListener('click',()=>document.getElementById('rdRaw')?.click());
  document.getElementById('rdClearM')?.addEventListener('click',()=>document.getElementById('rdClear')?.click());
  document.getElementById('rdCloseM')?.addEventListener('click',()=>{ rdHide(); menu?.classList.add('hidden'); });

  document.getElementById('rdClear')?.addEventListener('click', ()=>{
    const box=document.getElementById('controlTables'); if(box) box.innerHTML='';
    const raw=document.getElementById('controlStatus'); if(raw){ raw.textContent='—'; raw.classList.add('hidden'); }
    lastRaw=null;
  });
  document.getElementById('rdRaw')?.addEventListener('click', ()=>{
    const raw=document.getElementById('controlStatus');
    if(!raw) return;
    raw.classList.toggle('hidden');
    raw.textContent=typeof lastRaw==='string'?lastRaw:JSON.stringify(lastRaw,null,2);
  });
  document.getElementById('rdPos')?.addEventListener('change', (e)=>{
    rdState.position=e.target.value;
    rdApplyChrome(); rdPersistDisk();
  });
  document.getElementById('rdMode')?.addEventListener('change', (e)=>{
    rdState.mode=e.target.value;
    if(rdState.mode!=='inline') rdState.visible=true;
    rdApplyChrome(); rdPersistDisk();
  });
  document.querySelectorAll('[data-rdtab]').forEach(tab=>{
    tab.addEventListener('click', ()=>{
      document.querySelectorAll('[data-rdtab]').forEach(t=>t.classList.remove('on'));
      tab.classList.add('on');
      const out=document.getElementById('rdBodyOut');
      const hist=document.getElementById('rdBodyHist');
      if(tab.dataset.rdtab==='hist'){ out?.classList.add('hidden'); hist?.classList.remove('hidden'); rdRenderHist(); }
      else { hist?.classList.add('hidden'); out?.classList.remove('hidden'); }
    });
  });
  // drag resize
  const handle=document.getElementById('rdResize');
  const dock=document.getElementById('resultDock');
  if(handle && dock){
    let dragging=false, start=0, startSize=0;
    handle.addEventListener('mousedown', (ev)=>{
      dragging=true; startSize=rdState.size||220;
      start = rdState.position==='right' ? ev.clientX : ev.clientY;
      ev.preventDefault();
    });
    window.addEventListener('mousemove', (ev)=>{
      if(!dragging) return;
      let next=startSize;
      if(rdState.position==='bottom') next=startSize-(ev.clientY-start);
      else if(rdState.position==='top') next=startSize+(ev.clientY-start);
      else next=startSize-(ev.clientX-start);
      rdState.size=Math.max(120, Math.min(rdState.position==='right'?640:520, next));
      dock.style.setProperty('--rd-size', rdState.size+'px');
    });
    window.addEventListener('mouseup', ()=>{
      if(!dragging) return;
      dragging=false; rdPersistDisk();
    });
  }
}

// legacy raw button may be gone — guarded
document.getElementById('btnShowRaw')?.addEventListener('click', ()=>{
  const raw=document.getElementById('controlStatus');
  if(!raw) return;
  raw.classList.toggle('hidden');
  raw.textContent=typeof lastRaw==='string'?lastRaw:JSON.stringify(lastRaw,null,2);
});

// --- Control button definitions (max operator-session API) ---
const BTN = {
  overview:[
    ['health','b_health','GET','/health'],
    ['stack-status','b_stack','GET','/api/stack/status'],
    ['stack-apply','b_stack_apply','POST','/api/stack/apply','{}'],
    ['stack-rollback','b_stack_rb','POST','/api/stack/rollback','{}'],
    ['stack-wd','b_stack_wd','POST','/api/stack/watchdog','{}'],
    ['doctor','b_doctor','GET','/api/doctor'],
    ['domain','b_domain','GET','/api/domain'],
    ['bot','b_bot','GET','/api/bot-status'],
    ['status','b_status','GET','/api/status'],
    ['overview','b_overview','GET','/api/fleet/overview'],
    ['metrics','b_metrics','GET','/api/metrics'],
    ['metrics-hist','b_metrics_hist','GET','/api/metrics/history'],
    ['addons','b_addons','GET','/api/addons'],
    ['addons-lampac','b_lampac','GET','/api/addons/lampac'],
    ['sni','b_sni','GET','/api/sni'],
    ['sni-presets','b_sni_presets','GET','/api/sni-presets'],
    ['latest','b_latest','GET','/api/latest'],
    ['sessions','b_sessions','GET','/api/sessions'],
    ['gh-token','b_gh_token','GET','/api/update/github-token'],
    ['update-status','b_upd','GET','/api/update/status'],
    ['release-local','b_rel_local','GET','/api/release/local'],
    ['release-git-tags','b_rel_tags','GET','/api/release/git-tags'],
    ['release-mirror','b_rel_mirror','POST','/api/release/mirror-fetch','{}'],
    ['release-build','b_rel_build','POST','/api/release/build','{"tag":"","skip_darwin":true}'],
    ['release-build-st','Build status','GET','/api/release/build-status'],
    ['release-prune','Prune local','POST','/api/release/prune','{"keep":5}'],
  ],
  vpn:[
    ['vpn-users','b_vpn_users','GET','/vpn/users'],
    ['vpn-refresh','b_vpn_refresh','POST','/api/vpn/refresh-links','{}'],
    ['services-list','Services','GET','/api/services'],
    ['policy-apply','Policy apply','POST','/api/policy/apply','{}'],
  ],
  nodes:[
    ['nodes','b_nodes','GET','/api/nodes'],
    ['nodes-self','b_self','GET','/api/nodes/self'],
    ['secondary','b_sec_st','GET','/api/secondary/status'],
    ['secondary-links','b_sec_links','GET','/api/secondary/links'],
    ['ssh-hosts','b_ssh_hosts','GET','/api/ssh-hosts'],
    ['ssh-clear','b_ssh_clear','POST','/api/ssh-hosts/clear','{}'],
    ['mtls-certs','b_mtls','GET','/api/mtls/certs'],
    ['sites','b_sites','GET','/api/sites'],
  ],
  edge:[
    ['edge-pending','Pending','GET','/api/edge/pending'],
    ['edge-devices','Devices','GET','/api/edge/devices'],
    ['edge-metrics','Metrics','GET','/api/edge/metrics'],
    ['edge-metrics-hist','Metrics hist','GET','/api/edge/metrics/history'],
    ['edge-templates','Templates','GET','/api/edge/templates'],
    ['edge-backups','Backups','GET','/api/edge/backups'],
    ['edge-commands','Commands','GET','/api/edge/commands'],
    ['edge-results','Results','GET','/api/edge/results'],
    ['edge-guest-st','Guest status','GET','/api/edge/guest/status'],
  ],
  nvr:[
    ['nvr-cameras','Cameras','GET','/api/nvr/cameras'],
    ['nvr-cameras-add','Add camera','POST','/api/nvr/cameras','{"name":"cam1","lan_ip":"","rtsp_user":"","rtsp_password":"","cloud_password":"","rtsp_path":"/stream1","enabled":true,"record":true}'],
    ['nvr-cameras-del','Delete camera','POST','/api/nvr/cameras/delete','{"id":""}'],
    ['nvr-rec-start','Record start','POST','/api/nvr/recorder/start','{"id":""}'],
    ['nvr-rec-stop','Record stop','POST','/api/nvr/recorder/stop','{"id":""}'],
    ['nvr-ptz','PTZ','POST','/api/nvr/ptz','{"id":"","dir":"left"}'],
    ['nvr-clip','Clip token','POST','/api/nvr/clip/token','{"camera_id":"","path":""}'],
    ['nvr-motion','Motion','GET','/api/nvr/motion'],
    ['nvr-config','Config','GET','/api/nvr/config'],
    ['nvr-storage','Storage','GET','/api/nvr/storage'],
    ['nvr-events','Events','GET','/api/nvr/events'],
    ['nvr-segments','Segments','GET','/api/nvr/segments'],
    ['nvr-retention','Retention run','POST','/api/nvr/retention/run','{}'],
    ['nvr-go2rtc','go2rtc write','POST','/api/nvr/go2rtc','{}'],
  ],
  git:[
    ['git-repos','Repos','GET','/api/git/repos'],
    ['git-projects','Projects','GET','/api/git/projects'],
    ['git-project-queue','Mac queue','GET','/api/git/projects/queue?pending=1'],
    ['git-ship-jobs','Ship jobs','GET','/api/git/projects/ship-jobs'],
    ['build-host-status','Build host','GET','/api/build-host/status'],
    ['release-local-g','Local releases','GET','/api/release/local'],
    ['release-tags-g','Git tags','GET','/api/release/git-tags'],
    ['release-mirror-g','Mirror fetch (+auto build)','POST','/api/release/mirror-fetch','{}'],
    ['release-build-g','Build local tag','POST','/api/release/build','{"tag":"","skip_darwin":true}'],
    ['git-pipelines','Pipelines','GET','/api/git/pipelines'],
    ['git-artifacts','Artifacts','GET','/api/git/artifacts'],
    ['reg-status','Registry status','GET','/api/registry/status'],
    ['reg-catalog','Registry catalog','GET','/api/registry/catalog'],
    ['reg-ensure','Registry ensure','POST','/api/registry/ensure','{}'],
    ['reg-stop','Registry stop','POST','/api/registry/stop','{}'],
    ['reg-crane','Ensure crane','POST','/api/registry/crane','{}'],
  ],
  dns:[
    ['dns-lists','DNS lists','GET','/api/dns/lists'],
    ['dns-reload','DNS reload','POST','/api/dns/reload','{}'],
  ],
  backup:[
    ['backup-peer','Peer','GET','/api/backup/peer'],
    ['backup-run','Run now','POST','/api/backup/run','{}'],
    ['backup-verify','Verify latest','POST','/api/backup/verify','{}'],
    ['backup-verify-timer','Verify timer','POST','/api/backup/verify-install','{}'],
    ['backup-schedule','Schedule','GET','/api/backup/schedule'],
    ['backup-list','Files','GET','/api/backup/list'],
    ['sec-export','Secondary export','GET','/api/secondary/export'],
  ],
  probes:[
    ['probes','Probes','GET','/api/probes'],
    ['probes-cfg','Probes config','GET','/api/probes/config'],
    ['probes-uptime','Uptime','GET','/api/probes/uptime'],
    ['audit','Audit','GET','/api/audit'],
    ['devices','VPN devices','GET','/api/devices?refresh=1'],
    ['canary','Canary','GET','/api/canary'],
    ['smoke-dual','Smoke dual','GET','/api/smoke?mode=dual'],
    ['incident','Incident pack','POST','/api/incident/collect','{"hours":1}'],
    ['incident-tl','Incident timeline','GET','/api/incident/timeline'],
    ['presets','Policy presets','GET','/api/policy/presets'],
  ],
};




function wirePasswordToggles(){
  document.querySelectorAll('input[type="password"]').forEach(inp=>{
    if(inp.closest('.pwd-wrap')) return;
    const wrap=document.createElement('div');
    wrap.className='pwd-wrap';
    inp.parentNode.insertBefore(wrap, inp);
    wrap.appendChild(inp);
    const btn=document.createElement('button');
    btn.type='button';
    btn.className='pwd-toggle';
    btn.textContent = (uiLang==='ru') ? 'Показать' : 'Show';
    btn.addEventListener('click',()=>{
      const show = inp.type==='password';
      inp.type = show ? 'text' : 'password';
      btn.textContent = show
        ? ((uiLang==='ru')?'Скрыть':'Hide')
        : ((uiLang==='ru')?'Показать':'Show');
    });
    wrap.appendChild(btn);
  });
}

async function fillUpdateUI(){
  try{
    const st = await nodeFetch('/api/update/status');
    const rel = await nodeFetch('/api/update/releases');
    const line = document.getElementById('updStatusLine');
    const sel = document.getElementById('updTag');
    const badge = document.getElementById('updateBadge');
    if(line){
      const loc = st.local||'?';
      const lat = st.latest||'?';
      const msg = st.update_available ? (t('upd_available')||'update available') : (t('upd_ok')||'up to date');
      line.textContent = (t('upd_local')||'Local')+': '+loc+' · '+(t('upd_latest')||'Latest')+': '+lat+' · '+msg;
    }
    if(sel && rel && rel.releases){
      sel.innerHTML='';
      for(const r of rel.releases){
        const o=document.createElement('option');
        o.value=r.tag||r.Tag||'';
        o.textContent=o.value+(r.prerelease?' (pre)':'');
        sel.appendChild(o);
      }
    }
    if(badge){
      if(st && st.update_available){
        badge.textContent = '⬆ '+(st.latest||'');
        badge.classList.remove('hidden');
      } else {
        badge.textContent='';
        badge.classList.add('hidden');
      }
    }
  }catch(e){}
}

async function loadCatalogButtons(){
  try{
    const res = await fetch('/v1/catalog',{headers:{'X-Netductor-Token':ND_TOKEN}});
    if(!res.ok) return;
    const j = await res.json();
    window.__ndGroups = j.groups || [];
    window.__ndByGroup = j.by_group || {};
    if(j.by_section){
      for(const [sec, list] of Object.entries(j.by_section)){
        if(!BTN[sec]) BTN[sec]=[];
        BTN[sec] = list.map(a=>[a.id, (uiLang==='ru'?a.label_ru:a.label_en)||a.label_en, a.method, a.path, a.body||'']);
      }
    }
    // map group → merged button lists from sections
    for(const g of (j.groups||[])){
      const list = (j.by_group && j.by_group[g.id]) || [];
      BTN[g.id] = list.map(a=>[a.id, (uiLang==='ru'?a.label_ru:a.label_en)||a.label_en, a.method, a.path, a.body||'']);
    }
    renderControlNav();
  }catch(e){}
}
function renderControlNav(){
  const nav = document.getElementById('controlSub');
  if(!nav) return;
  const groups = window.__ndGroups || [];
  if(!groups.length){
    // fallback static
    nav.innerHTML = ['home','users','fleet','edge','media','data','adv'].map((id,i)=>
      `<button type="button" data-csec="${id}" class="${i===0?'active':''}">${id}</button>`).join(' ');
  } else {
    nav.innerHTML = groups.map((g,i)=>{
      const lab = uiLang==='ru' ? (g.label_ru||g.label_en) : (g.label_en||g.id);
      return `<button type="button" data-csec="${g.id}" class="${i===0?'active':''}">${lab}</button>`;
    }).join(' ');
  }
  nav.querySelectorAll('button').forEach(btn=>{
    btn.addEventListener('click',()=>{
      nav.querySelectorAll('button').forEach(b=>b.classList.remove('active'));
      btn.classList.add('active');
      document.querySelectorAll('.csec').forEach(el=>el.classList.add('hidden'));
      const panel = document.getElementById('csec-'+btn.dataset.csec);
      if(panel) panel.classList.remove('hidden');
      if(btn.dataset.csec==='home') try{ fillUpdateUI(); }catch(e){}
    });
  });
}
function groupOperateActions(list){
  // Heuristic buckets so Operate is cards, not a wall of primary buttons.
  const buckets=[
    {key:'diag', title: uiLang==='ru'?'Диагностика':'Diagnostics', match:/health|doctor|status|overview|metrics|attention|latest|sessions|domain|bot/i},
    {key:'stack', title: uiLang==='ru'?'Стек':'Stack', match:/^stack|watchdog/i},
    {key:'upd', title: uiLang==='ru'?'Обновления':'Updates', match:/update|release|rel_|gh-token|github/i},
    {key:'vpn', title: 'VPN / policy', match:/vpn|policy|services/i},
    {key:'nodes', title: uiLang==='ru'?'Узлы':'Nodes', match:/node|secondary|ssh|mtls|channel|path/i},
    {key:'edge', title: 'Edge', match:/edge|guest|site/i},
    {key:'nvr', title: 'NVR', match:/nvr|camera|clip|motion|go2rtc|ptz|record|segment|retention/i},
    {key:'git', title: 'Git / build', match:/git|build|registry|ship|pipeline|artifact|mirror/i},
    {key:'dns', title: 'DNS', match:/dns|blocky|sni/i},
    {key:'bak', title: uiLang==='ru'?'Бэкапы':'Backup', match:/backup|recover|drill/i},
    {key:'addon', title: uiLang==='ru'?'Сервисы':'Addons', match:/addon|lampac/i},
    {key:'misc', title: uiLang==='ru'?'Прочее':'Other', match:/.*/},
  ];
  const used=new Set();
  const out=[];
  for(const b of buckets){
    const items=[];
    for(const row of list){
      const id=row[0]||'';
      if(used.has(id)) continue;
      if(b.match.test(id) || b.match.test(String(row[1]||''))){
        items.push(row); used.add(id);
      }
    }
    if(items.length) out.push({title:b.title, items});
  }
  return out;
}
// Static BTN keys → shell panel ids (catalog groups may already use home/users/…)
const BTN_SEC_TO_PANEL={
  overview:'home', home:'home', updates:'home',
  vpn:'users', users:'users',
  nodes:'fleet', fleet:'fleet',
  edge:'edge',
  nvr:'media', media:'media',
  git:'data', backup:'data', dns:'data', probes:'data', data:'data',
  adv:'adv',
};
function mountButtons(){
  // clear panels first (several BTN keys may map to the same shell panel)
  const panelIds=new Set(Object.values(BTN_SEC_TO_PANEL));
  panelIds.forEach(id=>{ const el=document.getElementById('csec-'+id); if(el && id!=='adv') el.innerHTML=''; });
  for(const [sec, list] of Object.entries(BTN)){
    const panelId=BTN_SEC_TO_PANEL[sec]||sec;
    const el=document.getElementById('csec-'+panelId); if(!el) continue;
    // Group into calm op-cards (no primary-blue wall).
    const groups=groupOperateActions(list||[]);
    let h='<div class="op-grid" data-btn-sec="'+esc(sec)+'">';
    for(const g of groups){
      h+='<div class="op-card"><h3>'+g.title+'</h3><div class="op-links">';
      for(const [id,label,method,path,body] of g.items){
        const lab=(label&&label.indexOf('b_')===0)?t(label):label;
        const emph=/apply|doctor|backup.?now|ensure/i.test(id)||/apply|doctor/i.test(String(lab))? ' emph':'';
        h+='<button class="op-link'+emph+'" type="button" data-act="'+id+'">'+lab+'</button>';
      }
      h+='</div></div>';
    }
    h+='</div>';
    // append forms for complex POSTs
        if(sec==='updates' || sec==='home' || sec==='overview'){
      h+=`<div class="card" style="margin-top:.5rem"><h3>${t('tab_updates')||'Updates'}</h3>
      <p class="note" id="updStatusLine">—</p>
      <div class="row"><div><label>${t('upd_pick')||'Release'}</label><select id="updTag"></select></div>
      <div style="align-self:end"><button class="primary" type="button" data-act="upd-refresh">${t('upd_refresh')||'Refresh'}</button>
      <button class="primary" type="button" data-act="upd-apply">${t('upd_apply')||'Apply'}</button></div></div>
      <p class="note">Pre-backup + backup_pull wait on apply. Edge: use Edge → cmd agent_update.</p></div>`;
    }
    if(sec==='dns' || sec==='data'){
      h+=`<div class="row" style="margin-top:.75rem"><div><label>${t('l_dns_id')}</label><input id="dnsId" placeholder="adguard"/></div>
      <div><label>${t('l_dns_en')}</label><select id="dnsOn"><option value="1">${t('b_dns_on')}</option><option value="0">${t('b_dns_off')}</option></select></div></div>
      <button class="primary" type="button" data-act="dns-set">${t('b_dns_set')}</button>
      <p class="muted">DNS path: client → VPN → blocky (ads/YouTube filtering on PROXY) vs direct system DNS (DIRECT). See doctor check dns-path.</p>`;
    }
    if(sec==='backup'){
      h+=`<div class="row" style="margin-top:.75rem"><div><label>${t('l_bak_hour')}</label><input id="bakHour" type="number" min="0" max="23" value="1"/></div>
      <div><label>${t('l_bak_min')}</label><input id="bakMin" type="number" min="0" max="59" value="0"/></div></div>
      <div class="row"><div><label>${t('l_bak_tz')}</label><select id="bakUtc"><option value="1">UTC</option><option value="0">local</option></select></div><div></div></div>
      <button class="primary" type="button" data-act="backup-schedule-set">${t('b_bak_save')}</button>
      <p class="muted">Verify status and recover-drill age appear in Fleet overview (backup_verify, recover_drill_days). Run <code>netductor backup drill-mark</code> after a successful drill.</p>`;
    }
    if(sec==='vpn' || sec==='users'){
      h+=`<div class="row" style="margin-top:.75rem"><div><label>${t('l_vpn_name')}</label><input id="vpnName"/></div><div><label>${t('l_vpn_note')}</label><input id="vpnNote"/></div></div>
      <button class="primary" type="button" data-act="vpn-add">${t('b_vpn_add')}</button>
      <div class="row"><div><label>${t('l_vpn_act')}</label><input id="vpnActName"/></div><div></div></div>
      <button class="primary" type="button" data-act="vpn-enable">${t('b_vpn_enable')}</button>
      <button class="primary" type="button" data-act="vpn-disable">${t('b_vpn_disable')}</button>
      <button class="primary" type="button" data-act="vpn-revoke">${t('b_vpn_revoke')}</button>
      <button class="primary" type="button" data-act="vpn-link">${t('b_vpn_link')}</button>
      <div class="card" style="margin-top:.75rem" id="vpnPolicyCard">
        <h3>${t('l_vpn_policy')||'Access policy'}</h3>
        <p class="note">${t('l_vpn_policy_help')||'Load user, then edit checkboxes. Routers use Edge device id field below.'}</p>
        <div class="row"><div><label>${t('l_vpn_act')||'User'}</label><input id="vpnPolName" placeholder="alice"/></div>
        <div style="align-self:end"><button class="primary" type="button" data-act="vpn-policy-load">${t('b_vpn_pol_load')||'Load policy'}</button></div></div>
        <label><input type="checkbox" id="polInternet" checked/> ${t('l_pol_internet')||'Internet egress'}</label>
        <label style="margin-left:1rem"><input type="checkbox" id="polAll"/> ${t('l_pol_all')||'All internal services'}</label>
        <div id="polServices" style="margin:.5rem 0;display:flex;flex-wrap:wrap;gap:.5rem"></div>
        <button class="primary" type="button" data-act="vpn-policy-save">${t('b_vpn_pol_save')||'Save user policy'}</button>
        <button class="primary" type="button" data-act="vpn-policy-set-media">${t('b_vpn_pol_media')||'Preset: media'}</button>
        <button class="primary" type="button" data-act="vpn-policy-set-full">${t('b_vpn_pol_full')||'Preset: full'}</button>
        <button class="primary" type="button" data-act="vpn-policy-set-none">${t('b_vpn_pol_none')||'Preset: none'}</button>
        <div class="row" style="margin-top:.5rem"><div><label>${t('l_edge_pol')||'Edge device id'}</label><input id="edgePolId" placeholder="home-owrt-1"/></div>
        <div style="align-self:end"><button class="primary" type="button" data-act="edge-policy-load">${t('b_edge_pol_load')||'Load edge policy'}</button>
        <button class="primary" type="button" data-act="edge-policy-save">${t('b_edge_pol_save')||'Save edge policy'}</button></div></div>
      </div>`;
    }
    
    if(sec==='adv' || sec==='probes'){
      h+=`<h3>${t('ssh_title')||'SSH allowlist'}</h3>
      <p class="note">${t('ssh_note')||'Inventory → select fingerprints → Save (no live change) / Merge / Enforce (replaces authorized_keys). Probe secondary first.'}</p>
      <button class="primary" type="button" data-act="ssh-keys">${t('ssh_inv')||'Inventory'}</button>
      <button class="primary" type="button" data-act="ssh-probe-sec">${t('ssh_probe')||'Probe secondary'}</button>
      <button class="primary" type="button" data-act="ssh-allowlist-get">${t('ssh_status')||'Allowlist status'}</button>
      <label>${t('ssh_fps')||'Fingerprints (one per line, from inventory)'}</label>
      <textarea id="sshFpList" class="adv-body" rows="6" placeholder="SHA256:...."></textarea>
      <button class="primary" type="button" data-act="ssh-allowlist-save-form">${t('ssh_save')||'Save allowlist'}</button>
      <button class="primary" type="button" data-act="ssh-allowlist-merge-form">${t('ssh_merge')||'Merge'}</button>
      <button class="primary" type="button" data-act="ssh-allowlist-enforce-form">${t('ssh_enf')||'Enforce + secondary'}</button>
      <h3>${t('reboot_title')||'Reboot'}</h3>
      <p class="note">${t('reboot_note')||''}</p>
      <div class="row"><div><label>${t('reboot_target')||'Target'}</label><input id="rebootTarget" value="primary"/></div>
      <div style="align-self:end"><button class="primary" type="button" data-act="reboot-challenge-form">${t('reboot_challenge')||'Get code'}</button></div></div>
      <div class="row"><div><label>${t('reboot_code')||'Code'}</label><input id="rebootCode" placeholder="1234"/></div>
      <div style="align-self:end"><button class="primary" type="button" data-act="reboot-confirm-form">${t('reboot_confirm')||'Confirm reboot'}</button></div></div>
      <pre class="log" id="rebootChallengeOut"></pre>`;
    }

    if(sec==='nodes' || sec==='fleet'){
      h+=`<h3>Rooms</h3>
      <div class="row"><div><label>site_id</label><input id="roomSite" placeholder="home"/></div>
        <button type="button" class="primary" data-act="rooms-list">List rooms</button></div>
      <div id="roomsTable" class="note">—</div>
      <div class="row"><div><label>room id</label><input id="roomId" placeholder="kitchen"/></div>
        <div><label>name</label><input id="roomName" placeholder="Kitchen"/></div></div>
      <div class="row"><div><label>camera_ids (comma)</label><input id="roomCams"/></div>
        <div><label>notes</label><input id="roomNotes"/></div></div>
      <button type="button" class="primary" data-act="rooms-add">Upsert room</button>
      <button type="button" class="primary" data-act="rooms-del">Delete room</button>
      <div class="row"><div><label>photo file</label><input id="roomPhoto" type="file" accept="image/jpeg,image/png"/></div>
        <button type="button" class="primary" data-act="rooms-photo">Add photo (max 5)</button></div>
      <p class="muted">Photos: max 5 per room, ≤2 MiB jpeg/png. Session only.</p>`;
      h+=`<div class="row" style="margin-top:.75rem"><div><label>${t('l_hostname')}</label><input id="nodeHost"/></div><div><label>${t('l_node_id')}</label><input id="nodeId"/></div></div>
      <button class="primary" type="button" data-act="nodes-hostname">${t('b_hostname')}</button>
      <div class="row"><div><label>${t('l_svc')}</label><input id="nodeSvc" placeholder="sing-box"/></div><div><label>${t('l_journal')}</label><input id="nodeJournal"/></div></div>
      <button class="primary" type="button" data-act="nodes-restart">${t('b_svc_restart')}</button>
      <button class="primary" type="button" data-act="nodes-journal">${t('b_journal')}</button>
      <div class="row"><div><label>mTLS node_id</label><input id="mtlsNode"/></div><div></div></div>
      <button class="primary" type="button" data-act="mtls-rotate">Rotate cert</button>
      <button class="primary" type="button" data-act="mtls-revoke">Revoke cert</button>
      <div class="row"><div><label>site_id</label><input id="siteId"/></div><div><label>push rsc</label><input id="siteRsc"/></div></div>
      <button class="primary" type="button" data-act="sites-rsc">Get site RSC</button>
      <button class="primary" type="button" data-act="sites-push">Push RSC</button>
      <div class="row"><div><label>secondary cmd</label><input id="secCmd"/></div><div></div></div>
      <button class="primary" type="button" data-act="sec-cmd">Secondary cmd</button>
      <button class="primary" type="button" data-act="sec-sync">Secondary sync</button>
      <button class="primary" type="button" data-act="sec-exit">Secondary exit toggle info</button>`;
    }
    if(sec==='edge'){
      h+=`<div class="row" style="margin-top:.75rem"><div><label>device_id</label><input id="edgeId"/></div>
      <div><label>action</label><select id="edgeAction"><option>approve</option><option>deny</option><option>revoke</option></select></div></div>
      <button class="primary" type="button" data-act="edge-act">Submit</button>
      <div class="row"><div><label>guest device</label><input id="guestDid"/></div>
      <div><label>guest</label><select id="guestAct"><option value="status">status</option><option value="grant">grant</option><option value="revoke">revoke</option></select></div></div>
      <button class="primary" type="button" data-act="edge-guest">Guest</button>
      <div class="row"><div><label>recovery site</label><input id="edgeRecSite"/></div><div></div></div>
      <button class="primary" type="button" data-act="edge-recovery">Recovery code</button>
      <div class="row"><div><label>set-site device</label><input id="edgeSiteDid"/></div><div><label>site_id</label><input id="edgeSiteId"/></div></div>
      <button class="primary" type="button" data-act="edge-set-site">Set site</button>
      <div class="row"><div><label>template name</label><input id="edgeTpl"/></div><div><label>bind device</label><input id="edgeBindDid"/></div></div>
      <button class="primary" type="button" data-act="edge-template">Get template</button>
      <button class="primary" type="button" data-act="edge-bind">Bind template</button>
      <h3 style="margin-top:1rem">Template VPN / DNS</h3>
      <p class="muted">Edge template <code>vpn.dns</code> · mode · fallback (applies after device template apply)</p>
      <div class="row">
        <div><label>template id</label><input id="edgeVpnTpl" placeholder="default" value="default"/></div>
        <div><label>dns</label>
          <select id="edgeVpnDns">
            <option value="vpn">vpn (hijack via VLESS path)</option>
            <option value="wan">wan (ISP / router default)</option>
            <option value="off">off</option>
          </select>
        </div>
      </div>
      <div class="row">
        <div><label>${t('l_git_mig_mode')}</label>
          <select id="edgeVpnMode">
            <option value="tun">tun</option>
            <option value="off">off</option>
          </select>
        </div>
        <div><label>fallback</label>
          <select id="edgeVpnFallback">
            <option value="wan">wan (soft ISP)</option>
            <option value="block">block</option>
          </select>
        </div>
      </div>
      <div class="row">
        <div><label>soft_fallback</label>
          <select id="edgeVpnSoft">
            <option value="1">yes</option>
            <option value="0">no</option>
          </select>
        </div>
        <div><label>enabled</label>
          <select id="edgeVpnEn">
            <option value="1">yes</option>
            <option value="0">no</option>
          </select>
        </div>
      </div>
            <details><summary>Parameter help</summary>
      <ul>
        <li><b>enabled</b> — master switch for private Wi‑Fi VLESS client on apply</li>
        <li><b>dns=vpn</b> — hijack DNS; world resolvers via VLESS; .ru/.рф/.su → Yandex direct</li>
        <li><b>dns=wan</b> — do not force DNS through VLESS</li>
        <li><b>dns=off</b> — OpenWrt DNS defaults</li>
        <li><b>mode=tun</b> — TUN + auto_route for home LAN internet via VLESS</li>
        <li><b>mode=off</b> — no TUN policy from template</li>
        <li><b>fallback=wan</b> — if VLESS down, use ISP; <b>block</b> — no ISP fallback</li>
        <li><b>soft_fallback</b> — urltest so dead VLESS does not blackhole the house</li>
      </ul>
      <p class="muted">Save writes the template on primary; run template <b>Apply</b> on the device to push config.</p>
      </details>
      <button class="primary" type="button" data-act="edge-vpn-get">Load VPN settings</button>
      <button class="primary" type="button" data-act="edge-vpn-set">Save VPN settings</button>
      <div class="row"><div><label>cmd device</label><input id="edgeCmdDid"/></div><div><label>cmd</label><input id="edgeCmdName"/></div></div>
      <label>cmd arg</label><input id="edgeCmdArg"/>
      <button class="primary" type="button" data-act="edge-cmd">Enqueue cmd</button>
      <button class="primary" type="button" data-act="edge-export">Export</button>
      <button class="primary" type="button" data-act="edge-backup">Trigger backup</button>`;
    }
    if(sec==='nvr' || sec==='media'){
      h+=`<h4>Cameras</h4>
      <div class="row"><button type="button" class="primary" data-act="nvr-cam-refresh">Refresh table</button>
      <button type="button" class="primary" data-act="nvr-go2rtc">Write go2rtc.yaml</button></div>
      <div id="nvrCamTable" class="note">—</div>
      <h4>Add camera</h4>
      <div class="row"><div><label>name</label><input id="nvrAddName"/></div><div><label>LAN IP</label><input id="nvrAddIp"/></div></div>
      <div class="row"><div><label>RTSP user</label><input id="nvrAddUser" value=""/></div><div><label>RTSP password</label><input id="nvrAddPass" type="password"/></div></div>
      <div class="row"><div><label>cloud password (TPAP/talk)</label><input id="nvrAddCloud" type="password"/></div><div><label>path</label><input id="nvrAddPath" value="/stream1"/></div></div>
      <div class="row"><div><label>site_id</label><input id="nvrAddSite"/></div><div></div></div>
      <button type="button" class="primary" data-act="nvr-cam-add">Add / upsert</button>
      <h4>Wizard (DHCP lease → camera)</h4>
      <div class="row"><div><label>edge device_id</label><input id="nvrWizDid"/></div>
        <button type="button" class="primary" data-act="nvr-wiz-leases">1. Load leases</button></div>
      <pre class="log" id="nvrWizLeases" style="max-height:8rem"></pre>
      <div class="row"><div><label>mac</label><input id="nvrWizMac"/></div><div><label>ip from lease</label><input id="nvrWizIp"/></div></div>
      <div class="row"><div><label>name</label><input id="nvrWizName"/></div><div><label>RTSP password</label><input id="nvrWizPass" type="password"/></div></div>
      <button type="button" class="primary" data-act="nvr-wiz-add">2. Add from lease</button>
      <h4>Live</h4>
      <div class="row"><div><label>camera id</label><input id="nvrCamId"/></div>
        <button type="button" class="primary" data-act="nvr-live-info">Live info</button>
        <button type="button" class="primary" data-act="nvr-live-play">Play stream</button></div>
      <video id="nvrVideo" controls playsinline style="max-width:100%;max-height:320px;background:#111"></video>
      <div class="row"><div><label>PTZ</label><select id="nvrPtzDir"><option>left</option><option>right</option><option>up</option><option>down</option><option>stop</option></select></div>
        <button type="button" class="primary" data-act="nvr-ptz">PTZ</button>
        <button type="button" class="primary" data-act="nvr-rec-start">Record</button>
        <button type="button" class="primary" data-act="nvr-rec-stop">Stop</button>
        <button type="button" class="primary" data-act="nvr-cam-del">Delete</button></div>
      <p class="muted">Live uses session proxy to go2rtc on primary (localhost). Start go2rtc after writing yaml.</p>`;
    }
    if(sec==='git'){
      h+=`<div class="row" style="margin-top:.75rem"><div><label>${t('l_git_repo')}</label><input id="gitRepo" placeholder="my-repo"/></div>
      <div><label>${t('l_git_pipe')}</label><input id="gitPipe" placeholder="build.yml"/></div></div>
      <button class="primary" type="button" data-act="git-run">${t('b_git_run')}</button>
      <button class="primary" type="button" data-act="git-init">${t('b_git_init')}</button>
      <h4 style="margin:.75rem 0 .25rem">${t('l_git_proj')}</h4>
      <div class="row"><div><label>${t('l_git_proj_name')}</label><input id="gitProjName" placeholder="M4tg_bot"/></div>
      <div><label>${t('l_git_proj_up')}</label><input id="gitProjUp" placeholder="PavelNeyman/M4tg_bot"/></div></div>
      <div class="row"><div><label>${t('l_git_proj_wf')}</label><input id="gitProjWf" placeholder="ci/netductor.yml"/></div>
      <div><label>${t('l_git_proj_pipe')}</label><input id="gitProjPipe" placeholder="(optional pipeline)"/></div></div>
      <label><input type="checkbox" id="gitProjMac"/> host=mac (build on Mac, not VPS)</label>
      <label style="color:#c00"><input type="checkbox" id="gitProjOnFetch"/> ⚠️ build_on_fetch <b>OFF by default</b> — auto-build after every sync (heavy CPU/disk; enable only when needed)</label>
      <button class="primary" type="button" data-act="git-proj-list">${t('b_git_proj_list')}</button>
      <button class="primary" type="button" data-act="git-proj-add">${t('b_git_proj_add')}</button>
      <button class="primary" type="button" data-act="git-proj-detail">${t('b_git_proj_detail')}</button>
      <button class="primary" type="button" data-act="git-proj-sync">${t('b_git_proj_sync')}</button>
      <button class="primary" type="button" data-act="git-proj-build">Ship</button>
      <button type="button" data-act="git-ship-jobs">Ship jobs</button>
      <input id="gitShipJobId" placeholder="ship job id" style="min-width:12rem"/>
      <button type="button" data-act="git-ship-pause">Pause</button>
      <button type="button" data-act="git-ship-resume">Resume</button>
      <button type="button" data-act="git-ship-rm">Rm job</button>
      <div class="row"><div><label>${t('l_git_mig_owner')}</label><input id="gitMigOwner" placeholder="PavelNeyman" value="PavelNeyman"/></div>
        <div><label>mode</label><select id="gitMigMode"><option value="dry">dry-run</option><option value="sync">add+sync</option><option value="add">add only</option></select></div></div>
      <button class="primary" type="button" data-act="git-proj-migrate">${t('b_git_proj_migrate')}</button>
      <button class="primary" type="button" data-act="git-proj-sync-all">${t('b_git_proj_sync_all')}</button>
      <button class="primary" type="button" data-act="git-proj-build-all">Ship all</button>
      <button class="primary" type="button" data-act="git-proj-vps-queue">${t('b_git_proj_vps_queue')}</button>
      <div class="row"><div><label>vps job id</label><input id="gitVpsJobId" placeholder="build-…"/></div></div>
      <button type="button" data-act="git-proj-vps-cancel">${t('b_git_proj_vps_cancel')}</button>
      <button class="primary" type="button" data-act="git-proj-mirror-all">${t('b_git_proj_mirror_all')}</button>
      <p class="muted" style="margin:.5rem 0">CI cleanup policy: weekly timer <code>Sun 04:30 UTC</code> (docker prune + leftovers). Dry-run below.</p>
      <button type="button" data-act="cleanup-status">🧹 Cleanup status (dry-run)</button>
      <div class="row"><div><label>${t('l_git_repo')} (log/show)</label><input id="gitLogRepo" placeholder="my-repo"/></div>
      <div><label>${t('l_git_rev')}</label><input id="gitShowPath" placeholder="HEAD"/></div></div>
      <button class="primary" type="button" data-act="git-log">${t('b_git_log')}</button>
      <button class="primary" type="button" data-act="git-show">${t('b_git_show')}</button>
      <div class="row"><div><label>${t('l_git_art')}</label><input id="gitArtPath" placeholder="build/out.bin"/></div><div></div></div>
      <button class="primary" type="button" data-act="git-artifact">${t('b_git_art')}</button>`;
    }
    // append (do not overwrite other secs mapped to same panel)
    el.insertAdjacentHTML('beforeend', h);
  }
}
mountButtons();

const special = {
  'vpn-add': async()=>{ const name=document.getElementById('vpnName').value.trim(); const note=document.getElementById('vpnNote').value.trim(); if(!name) return {error:'name'}; return nodeFetch('/vpn/users',{method:'POST',body:JSON.stringify({name,note})}); },
  'vpn-enable': async()=>{ const n=document.getElementById('vpnActName').value.trim(); return nodeFetch('/vpn/users/'+encodeURIComponent(n)+'/enable',{method:'POST',body:'{}'}); },
  'vpn-disable': async()=>{ const n=document.getElementById('vpnActName').value.trim(); return nodeFetch('/vpn/users/'+encodeURIComponent(n)+'/disable',{method:'POST',body:'{}'}); },
  'vpn-revoke': async()=>{ const n=document.getElementById('vpnActName').value.trim(); return nodeFetch('/vpn/users/'+encodeURIComponent(n)+'/revoke',{method:'POST',body:'{}'}); },
  'vpn-link': async()=>{ const n=document.getElementById('vpnActName').value.trim(); return nodeFetch('/vpn/users/'+encodeURIComponent(n)+'/link'); },
  'vpn-policy-get': async()=>{
    const n=document.getElementById('vpnActName').value.trim();
    const pol=await nodeFetch('/vpn/users/'+encodeURIComponent(n)+'/policy');
    window.__ndPolicy=(pol && pol.policy)?pol.policy:pol;
    try{ const cat=await nodeFetch('/api/services'); window.__ndCatalog=cat; }catch(_){}
    renderPolicyCheckboxes();
    return pol;
  },
  'vpn-policy-save': async()=>{
    const n=document.getElementById('vpnActName').value.trim();
    if(!n) return {error:'name'};
    const body=collectPolicyForm();
    return nodeFetch('/vpn/users/'+encodeURIComponent(n)+'/policy',{method:'PUT',body:JSON.stringify(body)});
  },
  'edge-policy-save': async()=>{
    const id=document.getElementById('edgeId')?.value.trim()||document.getElementById('edge_pol_id')?.value.trim()||'';
    if(!id) return {error:'device_id'};
    const body=collectPolicyForm();
    return nodeFetch('/api/edge/device-policy',{method:'POST',body:JSON.stringify(Object.assign({device_id:id}, body))});
  },

  'vpn-policy-set-media': async()=>{
    const n=document.getElementById('vpnActName').value.trim();
    return nodeFetch('/vpn/users/'+encodeURIComponent(n)+'/policy',{method:'PUT',body:JSON.stringify({allow_internet:true,services:['lampac'],services_mode:'list'})});
  },
  'vpn-policy-set-full': async()=>{
    const n=document.getElementById('vpnActName').value.trim();
    return nodeFetch('/vpn/users/'+encodeURIComponent(n)+'/policy',{method:'PUT',body:JSON.stringify({allow_internet:true,services_mode:'all'})});
  },
  'vpn-policy-set-none': async()=>{
    const n=document.getElementById('vpnActName').value.trim();
    return nodeFetch('/vpn/users/'+encodeURIComponent(n)+'/policy',{method:'PUT',body:JSON.stringify({allow_internet:true,services:[],services_mode:'list'})});
  },
  'edge-policy-get': async()=>{
    const id=document.getElementById('edgeId').value.trim();
    const pol=await nodeFetch('/api/edge/device-policy?id='+encodeURIComponent(id));
    window.__ndPolicy=(pol && pol.policy)?pol.policy:pol;
    try{ const cat=await nodeFetch('/api/services'); window.__ndCatalog=cat; }catch(_){}
    renderPolicyCheckboxes();
    return pol;
  },

  'nodes-hostname': async()=>{ const hostname=document.getElementById('nodeHost').value.trim(); return nodeFetch('/api/nodes/hostname',{method:'POST',body:JSON.stringify({hostname,id:document.getElementById('nodeId').value.trim()})}); },
  'nodes-restart': async()=>{ return nodeFetch('/api/nodes/restart-service',{method:'POST',body:JSON.stringify({service:document.getElementById('nodeSvc').value.trim(),id:document.getElementById('nodeId').value.trim()})}); },
  'nodes-journal': async()=>{ const u=document.getElementById('nodeJournal').value.trim(); return nodeFetch('/api/nodes/journal'+(u?('?unit='+encodeURIComponent(u)):'')); },
  'mtls-rotate': async()=>{ const node_id=document.getElementById('mtlsNode').value.trim(); return nodeFetch('/api/mtls/rotate',{method:'POST',body:JSON.stringify({node_id})}); },
  'mtls-revoke': async()=>{ const node_id=document.getElementById('mtlsNode').value.trim(); return nodeFetch('/api/mtls/revoke',{method:'POST',body:JSON.stringify({node_id})}); },
  'sites-rsc': async()=>{ const id=document.getElementById('siteId').value.trim(); return nodeFetch('/api/sites/rsc'+(id?('?id='+encodeURIComponent(id)):'')); },
  'sites-push': async()=>{ return nodeFetch('/api/sites/push-rsc',{method:'POST',body:JSON.stringify({site_id:document.getElementById('siteId').value.trim(),rsc:document.getElementById('siteRsc').value})}); },
  'sec-cmd': async()=>{ return nodeFetch('/api/secondary/cmd',{method:'POST',body:JSON.stringify({cmd:document.getElementById('secCmd').value.trim()})}); },
  'sec-sync': async()=>{ return nodeFetch('/api/secondary/sync',{method:'POST',body:'{}'}); },
  'sec-exit': async()=>{ return nodeFetch('/api/secondary/exit'); },
  'edge-act': async()=>{ const id=document.getElementById('edgeId').value.trim(); const act=document.getElementById('edgeAction').value; return nodeFetch('/api/edge/'+act,{method:'POST',body:JSON.stringify({device_id:id})}); },
  'edge-guest': async()=>{ const did=document.getElementById('guestDid').value.trim(); const act=document.getElementById('guestAct').value; if(act==='status') return nodeFetch('/api/edge/guest/status'+(did?('?device_id='+encodeURIComponent(did)):'')); return nodeFetch('/api/edge/guest/'+act,{method:'POST',body:JSON.stringify({device_id:did})}); },
  'edge-recovery': async()=>{ return nodeFetch('/api/edge/recovery',{method:'POST',body:JSON.stringify({site_id:document.getElementById('edgeRecSite').value.trim()})}); },
  'edge-set-site': async()=>{ return nodeFetch('/api/edge/set-site',{method:'POST',body:JSON.stringify({device_id:document.getElementById('edgeSiteDid').value.trim(),site_id:document.getElementById('edgeSiteId').value.trim()})}); },
  'edge-template': async()=>{ const n=document.getElementById('edgeTpl').value.trim(); return nodeFetch('/api/edge/template'+(n?('?name='+encodeURIComponent(n)):'')); },
  'edge-bind': async()=>{ return nodeFetch('/api/edge/bind-template',{method:'POST',body:JSON.stringify({device_id:document.getElementById('edgeBindDid').value.trim(),template:document.getElementById('edgeTpl').value.trim()})}); },
  'edge-vpn-get': async()=>{
    const id=(document.getElementById('edgeVpnTpl').value||'default').trim()||'default';
    const j=await nodeFetch('/api/edge/templates/vpn?id='+encodeURIComponent(id));
    if(j&&j.vpn){
      const v=j.vpn;
      if(v.dns!=null) document.getElementById('edgeVpnDns').value=String(v.dns);
      else if(v.dns_mode!=null) document.getElementById('edgeVpnDns').value=String(v.dns_mode);
      if(v.mode!=null) document.getElementById('edgeVpnMode').value=String(v.mode);
      if(v.fallback!=null) document.getElementById('edgeVpnFallback').value=String(v.fallback);
      if(v.soft_fallback!=null) document.getElementById('edgeVpnSoft').value=(v.soft_fallback===true||v.soft_fallback==='true'||v.soft_fallback===1||v.soft_fallback==='1')?'1':'0';
      if(v.enabled!=null) document.getElementById('edgeVpnEn').value=(v.enabled===true||v.enabled==='true'||v.enabled===1||v.enabled==='1')?'1':'0';
    }
    return j;
  },
  'edge-vpn-set': async()=>{
    const id=(document.getElementById('edgeVpnTpl').value||'default').trim()||'default';
    return nodeFetch('/api/edge/templates/vpn',{method:'POST',body:JSON.stringify({
      id,
      dns: document.getElementById('edgeVpnDns').value,
      mode: document.getElementById('edgeVpnMode').value,
      fallback: document.getElementById('edgeVpnFallback').value,
      soft_fallback: document.getElementById('edgeVpnSoft').value==='1',
      enabled: document.getElementById('edgeVpnEn').value==='1'
    })});
  },
  'edge-cmd': async()=>{ return nodeFetch('/api/edge/cmd',{method:'POST',body:JSON.stringify({device_id:document.getElementById('edgeCmdDid').value.trim(),cmd:document.getElementById('edgeCmdName').value.trim(),arg:document.getElementById('edgeCmdArg').value})}); },
  'edge-export': async()=>{ return nodeFetch('/api/edge/export'); },
  'edge-backup': async()=>{ return nodeFetch('/api/edge/backup',{method:'POST',body:JSON.stringify({device_id:document.getElementById('edgeId').value.trim()})}); },
  'dns-set': async()=>{
    const id=document.getElementById('dnsId').value.trim();
    if(!id) return {error:'list id required'};
    const on=document.getElementById('dnsOn').value==='1';
    return nodeFetch('/api/dns/set',{method:'POST',body:JSON.stringify({id,enabled:on})});
  },
  'backup-schedule-set': async()=>{
    const hour=parseInt(document.getElementById('bakHour').value,10);
    const minute=parseInt(document.getElementById('bakMin').value,10);
    const utc=document.getElementById('bakUtc').value==='1';
    if(isNaN(hour)||hour<0||hour>23) return {error:'hour 0-23'};
    if(isNaN(minute)||minute<0||minute>59) return {error:'minute 0-59'};
    return nodeFetch('/api/backup/schedule',{method:'POST',body:JSON.stringify({hour,minute,utc})});
  },
  
  'nvr-cam-refresh': async()=>{
    const j=await nodeFetch('/api/nvr/cameras');
    const box=document.getElementById('nvrCamTable');
    if(!box) return j;
    const cams=(j&&j.cameras)||[];
    if(!cams.length){ box.innerHTML='<p class="muted">No cameras</p>'; return j; }
    let h='<table class="striped"><tr><th>id</th><th>name</th><th>ip</th><th>rec</th><th></th></tr>';
    cams.forEach(c=>{
      h+=`<tr><td><code>${c.id||''}</code></td><td>${c.name||''}</td><td>${c.lan_ip||''}</td><td>${c.record?'🟢':'—'}</td>`+
        `<td><button type="button" data-cam="${c.id}" class="nvr-pick">Select</button></td></tr>`;
    });
    h+='</table>';
    box.innerHTML=h;
    box.querySelectorAll('.nvr-pick').forEach(btn=>{
      btn.onclick=()=>{ const id=btn.getAttribute('data-cam'); const el=document.getElementById('nvrCamId'); if(el) el.value=id; };
    });
    return j;
  },
  'nvr-cam-add': async()=>{
    const body={
      name:document.getElementById('nvrAddName').value.trim(),
      lan_ip:document.getElementById('nvrAddIp').value.trim(),
      rtsp_user:document.getElementById('nvrAddUser').value.trim(),
      rtsp_password:document.getElementById('nvrAddPass').value,
      cloud_password:document.getElementById('nvrAddCloud').value,
      rtsp_path:document.getElementById('nvrAddPath').value.trim()||'/stream1',
      site_id:document.getElementById('nvrAddSite').value.trim(),
      enabled:true, record:true
    };
    return nodeFetch('/api/nvr/cameras',{method:'POST',body:JSON.stringify(body)});
  },
  'nvr-wiz-leases': async()=>{
    const did=document.getElementById('nvrWizDid').value.trim();
    if(!did) return {error:'device_id'};
    const pre=document.getElementById('nvrWizLeases');
    if(pre) pre.textContent='waiting for edge dhcp_leases…';
    const j=await nodeFetch('/api/nvr/site/leases',{method:'POST',body:JSON.stringify({device_id:did,wait:true,timeout_sec:90})});
    if(pre){
      const leases=(j&&j.leases)||[];
      if(!leases.length){ pre.textContent=JSON.stringify(j,null,2); return j; }
      let h='<table class="striped"><tr><th>host</th><th>ip</th><th>mac</th><th></th></tr>';
      leases.forEach((L,i)=>{
        const host=L.hostname||'—';
        h+=`<tr><td>${host}</td><td>${L.ip||''}</td><td><code>${L.mac||''}</code></td>`+
          `<td><button type="button" class="nvr-lease-pick" data-i="${i}">Use</button></td></tr>`;
      });
      h+='</table>';
      pre.innerHTML=h;
      pre.querySelectorAll('.nvr-lease-pick').forEach(btn=>{
        btn.onclick=()=>{
          const L=leases[+btn.getAttribute('data-i')];
          if(!L) return;
          const ip=document.getElementById('nvrWizIp'); if(ip) ip.value=L.ip||'';
          const mac=document.getElementById('nvrWizMac'); if(mac) mac.value=L.mac||'';
          const name=document.getElementById('nvrWizName');
          if(name && !name.value) name.value=(L.hostname&&L.hostname!=='*')?L.hostname:('cam-'+(L.mac||'').replace(/:/g,'').slice(-6));
        };
      });
    }
    return j;
  },
  'nvr-wiz-add': async()=>{
    const body={
      name:document.getElementById('nvrWizName').value.trim(),
      lan_ip:document.getElementById('nvrWizIp').value.trim(),
      mac:document.getElementById('nvrWizMac').value.trim(),
      rtsp_password:document.getElementById('nvrWizPass').value,
      site_id:document.getElementById('nvrWizDid').value.trim(),
      enabled:true, record:true, rtsp_path:'/stream1'
    };
    return nodeFetch('/api/nvr/cameras',{method:'POST',body:JSON.stringify(body)});
  },
  'nvr-live-info': async()=>{
    const id=document.getElementById('nvrCamId').value.trim();
    if(!id) return {error:'camera id'};
    return nodeFetch('/api/nvr/live?id='+encodeURIComponent(id));
  },
  'nvr-live-play': async()=>{
    const id=document.getElementById('nvrCamId').value.trim();
    if(!id) return {error:'camera id'};
    const v=document.getElementById('nvrVideo');
    if(v){ v.src='/api/nvr/stream?id='+encodeURIComponent(id); v.play().catch(()=>{}); }
    return {ok:true, stream:'/api/nvr/stream?id='+id};
  },
  
  'rooms-list': async()=>{
    const site=(document.getElementById('roomSite')?.value||'').trim();
    if(!site) return {error:'site_id'};
    const j=await nodeFetch('/api/sites/rooms?site='+encodeURIComponent(site));
    const box=document.getElementById('roomsTable');
    const rooms=(j&&j.rooms)||[];
    if(box){
      if(!rooms.length) box.innerHTML='<p class="muted">No rooms</p>';
      else {
        let h='<table class="striped"><tr><th>id</th><th>name</th><th>cams</th><th>photos</th><th></th></tr>';
        rooms.forEach(r=>{
          h+=`<tr><td><code>${r.id}</code></td><td>${r.name||''}</td><td>${(r.camera_ids||[]).join(', ')}</td><td>${r.photo_count||0}/5</td>`+
            `<td><button type="button" class="room-pick" data-id="${r.id}">Select</button></td></tr>`;
        });
        h+='</table>';
        box.innerHTML=h;
        box.querySelectorAll('.room-pick').forEach(btn=>{
          btn.onclick=()=>{ const el=document.getElementById('roomId'); if(el) el.value=btn.getAttribute('data-id'); };
        });
      }
    }
    return j;
  },
  'rooms-add': async()=>{
    const site=(document.getElementById('roomSite')?.value||'').trim();
    const id=(document.getElementById('roomId')?.value||'').trim();
    const name=(document.getElementById('roomName')?.value||'').trim();
    const cams=(document.getElementById('roomCams')?.value||'').split(',').map(s=>s.trim()).filter(Boolean);
    const notes=(document.getElementById('roomNotes')?.value||'').trim();
    return nodeFetch('/api/sites/rooms',{method:'POST',body:JSON.stringify({site_id:site,id,name,camera_ids:cams,notes})});
  },
  'rooms-del': async()=>{
    const site=(document.getElementById('roomSite')?.value||'').trim();
    const id=(document.getElementById('roomId')?.value||'').trim();
    return nodeFetch('/api/sites/rooms/delete',{method:'POST',body:JSON.stringify({site_id:site,id})});
  },
  'rooms-photo': async()=>{
    const site=(document.getElementById('roomSite')?.value||'').trim();
    const id=(document.getElementById('roomId')?.value||'').trim();
    const inp=document.getElementById('roomPhoto');
    if(!inp||!inp.files||!inp.files[0]) return {error:'choose file'};
    const file=inp.files[0];
    const buf=await file.arrayBuffer();
    const bytes=new Uint8Array(buf);
    let bin=''; for(let i=0;i<bytes.length;i++) bin+=String.fromCharCode(bytes[i]);
    const b64=btoa(bin);
    return nodeFetch('/api/sites/rooms/photo',{method:'POST',body:JSON.stringify({site_id:site,id,data_base64:b64,content_type:file.type||'image/jpeg'})});
  },
  'nvr-ptz': async()=>{ return nodeFetch('/api/nvr/ptz',{method:'POST',body:JSON.stringify({id:document.getElementById('nvrCamId').value.trim(),dir:document.getElementById('nvrPtzDir').value})}); },
  'nvr-rec-start': async()=>{ return nodeFetch('/api/nvr/recorder/start',{method:'POST',body:JSON.stringify({id:document.getElementById('nvrCamId').value.trim()})}); },
  'nvr-rec-stop': async()=>{ return nodeFetch('/api/nvr/recorder/stop',{method:'POST',body:JSON.stringify({id:document.getElementById('nvrCamId').value.trim()})}); },
  'nvr-cam-del': async()=>{ return nodeFetch('/api/nvr/cameras/delete',{method:'POST',body:JSON.stringify({id:document.getElementById('nvrCamId').value.trim()})}); },
  'nvr-leases': async()=>{ const did=document.getElementById('nvrSiteDid').value.trim(); return nodeFetch('/api/nvr/site/leases'+(did?('?device_id='+encodeURIComponent(did)):'')); },
  'nvr-wifi': async()=>{ const did=document.getElementById('nvrSiteDid').value.trim(); return nodeFetch('/api/nvr/site/wifi_clients'+(did?('?device_id='+encodeURIComponent(did)):'')); },
  'nvr-dhcp': async()=>{ return nodeFetch('/api/nvr/site/dhcp_static',{method:'POST',body:JSON.stringify({device_id:document.getElementById('dhcpDid').value.trim(),mac:document.getElementById('dhcpMac').value.trim(),ip:document.getElementById('dhcpIp').value.trim(),name:document.getElementById('dhcpName').value.trim()})}); },
  'nvr-clip-token': async()=>{ return nodeFetch('/api/nvr/clip/token',{method:'POST',body:JSON.stringify({path:document.getElementById('nvrClip').value.trim()})}); },
  'git-run': async()=>{ return nodeFetch('/api/git/pipeline',{method:'POST',body:JSON.stringify({repo:document.getElementById('gitRepo').value.trim(),pipeline:document.getElementById('gitPipe').value.trim()})}); },
  'git-init': async()=>{ const name=document.getElementById('gitRepo').value.trim(); if(!name) return {error:'repo name'}; return nodeFetch('/api/git/repos',{method:'POST',body:JSON.stringify({name})}); },
  'git-proj-list': async()=> nodeFetch('/api/git/projects'),
  'git-proj-detail': async()=>{ const name=(document.getElementById('gitProjName')?.value||'').trim(); if(!name) return {error:'name'}; return nodeFetch('/api/git/projects/detail?name='+encodeURIComponent(name)); },
  'git-proj-add': async()=>{ const name=document.getElementById('gitProjName').value.trim(); const upstream=document.getElementById('gitProjUp').value.trim();
    if(!name||!upstream) return {error:'name and upstream'}; const workflow=document.getElementById('gitProjWf').value.trim(); const pipeline=document.getElementById('gitProjPipe').value.trim();
    const host=document.getElementById('gitProjMac')&&document.getElementById('gitProjMac').checked?'mac':'vps'; const build_on_fetch=!!(document.getElementById('gitProjOnFetch')&&document.getElementById('gitProjOnFetch').checked); return nodeFetch('/api/git/projects',{method:'POST',body:JSON.stringify({name,upstream,workflow,pipeline,host,build_on_fetch})}); },
  'git-proj-sync': async()=>{ const name=document.getElementById('gitProjName').value.trim(); if(!name) return {error:'name'}; return nodeFetch('/api/git/projects/sync',{method:'POST',body:JSON.stringify({name})}); },
  'git-proj-build': async()=>{ const name=document.getElementById('gitProjName').value.trim(); if(!name) return {error:'name'}; return nodeFetch('/api/git/projects/ship',{method:'POST',body:JSON.stringify({name})}); },
  'git-proj-ship': async()=>{ const name=document.getElementById('gitProjName').value.trim(); if(!name) return {error:'name'}; return nodeFetch('/api/git/projects/ship',{method:'POST',body:JSON.stringify({name})}); },
  'git-ship-jobs': async()=> nodeFetch('/api/git/projects/ship-jobs'),
  'git-ship-pause': async()=>{ const id=(document.getElementById('gitShipJobId')?.value||'').trim(); if(!id) return {error:'id'}; return nodeFetch('/api/git/projects/ship-jobs',{method:'POST',body:JSON.stringify({action:'pause',id})}); },
  'git-ship-resume': async()=>{ const id=(document.getElementById('gitShipJobId')?.value||'').trim(); if(!id) return {error:'id'}; return nodeFetch('/api/git/projects/ship-jobs',{method:'POST',body:JSON.stringify({action:'resume',id})}); },
  'build-host-status': async()=> nodeFetch('/api/build-host/status'),
  'build-host-deploy': async()=>({ error: 'Deploy build-host only from operator (Mac TUI / netductor-op deploy build-host), not from node Web' }),
  'git-ship-rm': async()=>{ const id=(document.getElementById('gitShipJobId')?.value||'').trim(); if(!id) return {error:'id'}; return nodeFetch('/api/git/projects/ship-jobs',{method:'POST',body:JSON.stringify({action:'rm',id})}); },
  'git-proj-migrate': async()=>{
    const owner=(document.getElementById('gitMigOwner')?.value||'PavelNeyman').trim();
    const mode=(document.getElementById('gitMigMode')?.value||'dry');
    const dry_run=mode==='dry';
    const sync=mode==='sync';
    return nodeFetch('/api/git/projects/migrate',{method:'POST',body:JSON.stringify({owner,dry_run,sync})});
  },
  'git-proj-sync-all': async()=> nodeFetch('/api/git/projects/sync-all',{method:'POST',body:JSON.stringify({})}),
  'git-proj-build-all': async()=> nodeFetch('/api/git/projects/build-all',{method:'POST',body:JSON.stringify({include_mac:false})}),
  'git-proj-vps-queue': async()=> nodeFetch('/api/git/projects/vps-queue?pending=0'),
  'git-proj-vps-cancel': async()=>{ const id=(document.getElementById('gitVpsJobId')?.value||'').trim(); if(!id) return {error:'id'}; return nodeFetch('/api/git/projects/vps-queue',{method:'POST',body:JSON.stringify({action:'cancel',id})}); },

  'git-proj-mirror-all': async()=> nodeFetch('/api/git/projects/mirror-push',{method:'POST',body:JSON.stringify({all:true})}),
  'cleanup-status': async()=> nodeFetch('/api/cleanup'),



  'git-log': async()=>{ const r=document.getElementById('gitLogRepo').value.trim()||document.getElementById('gitRepo').value.trim(); return nodeFetch('/api/git/log'+(r?('?name='+encodeURIComponent(r)):'')); },
  'git-show': async()=>{
    const r=document.getElementById('gitLogRepo').value.trim()||document.getElementById('gitRepo').value.trim();
    const rev=document.getElementById('gitShowPath').value.trim()||'HEAD';
    return nodeFetch('/api/git/show?name='+encodeURIComponent(r)+'&rev='+encodeURIComponent(rev));
  },
  'git-artifact': async()=>{
    const path=document.getElementById('gitArtPath').value.trim();
    return nodeFetch('/api/git/artifact'+(path?('?path='+encodeURIComponent(path)):''));
  },
  'upd-refresh': async()=>{
    await fillUpdateUI();
    return {ok:true, refreshed:true};
  },
  'upd-apply': async()=>{
    const tag=(document.getElementById('updTag')||{}).value||'';
    if(!confirm('Apply update '+(tag||'latest')+' on primary?')) return {cancelled:true};
    return nodeFetch('/api/update/apply',{method:'POST',body:JSON.stringify({version:tag,component:'node'})});
  },
  'adv-send': async()=>{
    const method=document.getElementById('advMethod').value;
    let path=document.getElementById('advPath').value.trim(); if(!path.startsWith('/')) path='/'+path;
    const body=document.getElementById('advBody').value;
    if(method==='POST' && !confirm('POST '+path+' ?')) return {cancelled:true};
    return nodeFetch(path,{method, body: method==='POST'?body:undefined});
  },
};

// wire simple BTN acts
for(const list of Object.values(BTN)){
  for(const [id,label,method,path,body] of list){
    special[id]=special[id]||(async()=>nodeFetch(path,{method, body: method==='POST'?(body||'{}'):undefined}));
  }
}

// Delegate from body: buttons may live in main-control or re-mounted panels
document.body.addEventListener('click', async (ev)=>{
  const btn=ev.target.closest('[data-act]'); if(!btn) return;
  // ignore pure UI toggles without special handler path
  const act=btn.dataset.act;
  if(!act || !special[act]){
    if(act && btn.closest('#main-control')){
      rdLastAct=act; rdLastBtn=btn;
      showControl({ok:false,status:0,data:{error:'no handler for '+act}}, {act});
    }
    return;
  }
  // avoid double-fire with other handlers on same click when not operate
  if(!btn.closest('#main-control') && !btn.closest('.op-card')) return;
  rdLastAct=act; rdLastBtn=btn;
  showControl({loading:act}, {act});
  try{ showControl(await special[act](), {act}); }catch(e){ showControl({error:String(e)}, {act}); }
});

document.body.addEventListener('click', async (ev)=>{
  const t=ev.target.closest('.rowact'); if(!t) return;
  try{
    if(t.dataset.kind==='vpn-disable') showControl(await nodeFetch('/vpn/users/'+encodeURIComponent(t.dataset.name)+'/disable',{method:'POST',body:'{}'}));
    if(t.dataset.kind==='vpn-enable') showControl(await nodeFetch('/vpn/users/'+encodeURIComponent(t.dataset.name)+'/enable',{method:'POST',body:'{}'}));
    if(t.dataset.kind==='edge-approve') showControl(await nodeFetch('/api/edge/approve',{method:'POST',body:JSON.stringify({device_id:t.dataset.id})}));
    if(t.dataset.kind==='edge-deny') showControl(await nodeFetch('/api/edge/deny',{method:'POST',body:JSON.stringify({device_id:t.dataset.id})}));
  }catch(e){ showControl(String(e)); }
});

document.getElementById('btnTunnelStart').onclick=async()=>{ showControl(await ensureTunnel()); };
document.getElementById('btnTunnelStop').onclick=async()=>{ await fetch('/v1/tunnel/stop',{method:'POST',headers:{'X-Netductor-Token':ND_TOKEN}}); showControl(await refreshTunnelBadge()); };
document.getElementById('btnSessionIssue').onclick=async()=>{
  const s=settings();
  const res=await fetch('/v1/session/issue',{method:'POST',headers:{'Content-Type':'application/json','X-Netductor-Token':ND_TOKEN},
    body:JSON.stringify({host:s.primary_host,user:s.user||'root',key:s.key,hours:'72'})});
  const j=await res.json();
  if(j.token){ saveSettingsObj(Object.assign({},s,{node_session:j.token})); showControl({ok:true,saved:j.saved}); }
  else showControl(j);
};
document.getElementById('btnSessionLoad').onclick=async()=>{
  const res=await fetch('/v1/session/local',{headers:{'X-Netductor-Token':ND_TOKEN}});
  const j=await res.json();
  if(j.token){ saveSettingsObj(Object.assign({},settings(),{node_session:j.token})); loadSettingsForm(); }
  showControl(j);
};

['form-fleet','form-primary','form-build','form-secondary','form-edge','form-edge-luci','form-site','form-nvr','form-creds'].forEach(id=>{ const f=document.getElementById(id); if(f) loadForm(id.replace('form-',''),f); });
prefillAllFromConfig(); loadNetmap(); wirePasswordToggles(); syncEdgePanels();
document.getElementById('langEn').onclick=()=>{uiLang='en';localStorage.setItem('nd_op_lang','en');applyI18n();loadNetmap()};
document.getElementById('langRu').onclick=()=>{uiLang='ru';localStorage.setItem('nd_op_lang','ru');applyI18n();loadNetmap()};
applyI18n();
(async()=>{ try{ const res=await fetch('/v1/session/local',{headers:{'X-Netductor-Token':ND_TOKEN}}); const j=await res.json(); if(j.ok&&j.token){ const s=settings(); if(!s.node_session){ s.node_session=j.token; saveSettingsObj(s);} } }catch(e){} })();
fetch('/v1/meta').then(r=>r.json()).then(m=>{ document.getElementById('metaLine').textContent='op v'+m.version; }).catch(()=>{});
updateTunnelHint();


async function refreshGhTokenStatus(){
  const el=document.getElementById('set_github_token_status');
  if(!el) return;
  try{
    const r=await nodeFetch('/api/update/github-token');
    const d=r.data||{};
    if(d.configured) el.value='✅ '+(d.hint||'')+' ('+(d.source||'')+')';
    else el.value='❌ not set';
  }catch(e){ el.value='error: '+e; }
}
(function(){
  const save=document.getElementById('btnGhTokenSave');
  const clr=document.getElementById('btnGhTokenClear');
  const ref=document.getElementById('btnGhTokenRefresh');
  if(save) save.onclick=async()=>{
    const tok=(document.getElementById('set_github_token')||{}).value||'';
    if(!tok.trim()){ alert('token required'); return; }
    const kind=(document.getElementById('ghTokKind')||{}).value||'releases'; const r=await nodeFetch('/api/update/github-token',{method:'POST',body:JSON.stringify({token:tok.trim(),kind})});
    if(!r.ok) alert(JSON.stringify(r.data));
    document.getElementById('set_github_token').value='';
    await refreshGhTokenStatus();
  };
  if(clr) clr.onclick=async()=>{
    await nodeFetch('/api/update/github-token',{method:'POST',body:JSON.stringify({clear:true})});
    await refreshGhTokenStatus();
  };
  if(ref) ref.onclick=()=>refreshGhTokenStatus();
  // refresh when opening settings
  document.querySelectorAll('#mainTabs button').forEach(b=>{
    b.addEventListener('click',()=>{ if(b.dataset.main==='settings') refreshGhTokenStatus(); });
  });
})();

document.getElementById('form-edge-luci')?.addEventListener('submit', async e=>{
  const fd0=new FormData(e.target); saveForm('edge-luci',fd0);
  e.preventDefault();
  const fd=new FormData(e.target);
  const s=typeof settings==='function'?settings():{};
  const body={
    router_host:fd.get('router_host'), router_password:fd.get('router_password')||'',
    primary_key:fd.get('primary_key')||s.key||'',
    device_id:fd.get('device_id')||'',
    action:fd.get('action'), hours:parseFloat(fd.get('hours')||'1'), via:fd.get('via')||'ssh'
  };
  try{
    const r=await fetch('/v1/edge/luci',{method:'POST',headers:{'Content-Type':'application/json','X-Netductor-Token':ND_TOKEN},body:JSON.stringify(body)});
    const j=await r.json().catch(()=>({}));
    log(r.ok?JSON.stringify(j,null,2):(j.error||JSON.stringify(j)||r.statusText));
  }catch(err){ log(String(err)); }
});


document.getElementById('btnEdgeOfflinePrep')?.addEventListener('click', async ()=>{
  const form=document.getElementById('form-edge');
  if(!form) return;
  const fd=new FormData(form);
  const isRu=(document.documentElement.lang||localStorage.getItem('nd_lang')||'').startsWith('ru');
  const force=!!window.__ndOfflineForceRefresh;
  window.__ndOfflineForceRefresh=false;
  if(!confirm(isRu
    ? (force
      ? 'Полное обновление кэша: интернет + SSH primary :52222. Перекачать все agent?'
      : 'Нужен интернет и SSH на primary (:52222). Скачаем token и недостающие agent (уже в кэше не трогаем). Продолжить?')
    : (force
      ? 'Full cache refresh: internet + SSH primary :52222. Re-download all agents?'
      : 'Needs internet + SSH to primary (:52222). Downloads token + missing agents only (cached kept). Continue?'))) return;
  const btn=document.getElementById('btnEdgeOfflinePrep');
  if(btn) btn.disabled=true;
  try{
    const text = await streamPost('/v1/edge/offline-prep',{
      primary_host:fd.get('primary_host'), primary_key:fd.get('primary_key'),
      primary_user:fd.get('primary_user')||'root', key_passphrase:fd.get('key_passphrase')||'',
      device_id:fd.get('device_id')||'',
      force_refresh: force
    }, btn);
    const off=document.getElementById('edge_offline');
    if(off) off.checked=true;
    const body = (typeof text==='string'?text:'') + (document.getElementById('log')?.textContent||'');
    const partial = /offline-prep PARTIAL|WARN offline pack incomplete/i.test(body);
    const okFull = /offline-prep ok /i.test(body) && !partial;
    if(btn){
      if(partial){
        btn.textContent = isRu ? 'Докачать недостающее' : 'Retry missing downloads';
        btn.classList.add('warn');
        alert(isRu
          ? 'Офлайн-пакет частичный: не все agent/arch скачались.\n\nНажмите «Докачать недостающее» — подтянутся только пробелы (кэш сохранится).\nДля Cudy обычно достаточно mipsle.'
          : 'Offline pack is PARTIAL: some agent arches failed.\n\nClick «Retry missing downloads» to fetch only gaps (cache kept).\nCudy usually only needs mipsle.');
      } else if(okFull){
        btn.textContent = isRu ? 'Подготовить offline pack' : 'Prepare offline pack';
        btn.classList.remove('warn');
      }
    }
  } finally { if(btn) btn.disabled=false; }
});

function collectPolicyForm(){
  const internet=!!(document.getElementById('polInternet')||{}).checked;
  const all=!!(document.getElementById('polAll')||{}).checked;
  const services=[];
  document.querySelectorAll('#polServices input[type=checkbox][data-sid]').forEach(cb=>{ if(cb.checked) services.push(cb.dataset.sid); });
  return {allow_internet:internet, services_mode: all?'all':'list', services};
}
function renderPolicyCheckboxes(){
  const box=document.getElementById('polServices');
  if(!box) return;
  const cat=window.__ndCatalog||{};
  const pol=window.__ndPolicy||{};
  const services=((cat.services)||(cat.catalog&&cat.catalog.services)||[]).filter(s=>s && s.kind!=='egress' && !s.disabled);
  const mode=(pol.services_mode||'list');
  const allowed=new Set(pol.services||[]);
  const inet=document.getElementById('polInternet');
  const all=document.getElementById('polAll');
  if(inet) inet.checked=pol.allow_internet!==false;
  if(all) all.checked=mode==='all';
  box.innerHTML = services.map(s=>{
    const on = mode==='all' || allowed.has(s.id);
    return `<label style="min-width:9rem"><input type="checkbox" data-sid="${s.id}" ${on?'checked':''}/> ${s.title||s.id}</label>`;
  }).join('') || '<span class="note">No internal services in catalog</span>';
}




/* ===== Edge modular wizard (probe → card → modules → details) ===== */
window.__edgePreview = null;

function edgeSetStep(n){
  document.querySelectorAll('#edge_steps .estep').forEach(el=>{
    el.classList.toggle('active', Number(el.dataset.step)===n);
  });
  ['connect','card','modules','details'].forEach((name,i)=>{
    const blk=document.getElementById('edge_blk_'+name);
    if(blk) blk.classList.toggle('hidden', i!==n-1);
  });
}

function edgeCollectModules(){
  const mods={};
  document.querySelectorAll('#edge_mod_list input[data-mod]').forEach(inp=>{
    mods[inp.dataset.mod]=!!inp.checked;
  });
  return mods;
}

function edgeIntentFromModules(mods){
  const net = !!(mods.lan_baseline || mods.wan_baseline || mods.wifi_ap);
  const guest = !!mods.guest;
  const netEl=document.getElementById('edge_net_cfg');
  const gEl=document.getElementById('edge_guest');
  if(netEl) netEl.value = net ? 'on' : '';
  if(gEl) gEl.checked = guest;
  return {net_configure:net, guest_enable:guest};
}

function edgeRenderCard(resp){
  const c=resp.card||{};
  const model=document.getElementById('edge_card_model');
  const meta=document.getElementById('edge_card_meta');
  const badge=document.getElementById('edge_card_preset');
  const chips=document.getElementById('edge_card_chips');
  const stats=document.getElementById('edge_card_stats');
  if(model) model.textContent = c.model || c.board || 'OpenWrt device';
  if(meta) meta.textContent = [c.serial&&('S/N '+c.serial), c.arch, c.os].filter(Boolean).join(' · ');
  if(badge) badge.textContent = (resp.preset||resp.suggested_preset||'') + (resp.suggested_preset&&resp.preset!==resp.suggested_preset ? ' (suggested '+resp.suggested_preset+')' : '');
  if(chips){
    chips.innerHTML='';
    const add=(label,on,warn)=>{
      const s=document.createElement('span');
      s.className='edge-chip'+(on?' on':'')+(warn?' warn':'');
      s.textContent=label;
      chips.appendChild(s);
    };
    add('LAN '+(c.lan_ip||'—'), !!c.lan_ip);
    add(c.wan_capable ? t('edge_chip_wan_yes') : t('edge_chip_wan_no'), c.wan_capable, !c.wan_capable);
    add(t('edge_chip_eth')+(c.eth_count||0), c.eth_count>0);
    (c.bands||[]).forEach(b=>add('Wi‑Fi '+b, true));
    if(!(c.bands||[]).length) add(t('edge_chip_no_radio'), false, true);
    (c.ssid_now||[]).forEach(s=>add('SSID '+s, true));
  }
  if(stats){
    const mem = c.mem_total_kb ? Math.round((c.mem_avail_kb||0)/1024)+' / '+Math.round(c.mem_total_kb/1024)+' MiB' : '—';
    const parts=[];
    const fmtMi=k=>k?Math.round(k/1024)+' MiB':'';
    if(c.overlay_total_kb) parts.push(t('edge_storage_overlay')+' '+fmtMi(c.overlay_free_kb)+' / '+fmtMi(c.overlay_total_kb));
    else if(c.root_total_kb) parts.push(t('edge_storage_root')+' '+fmtMi(c.root_free_kb)+' / '+fmtMi(c.root_total_kb));
    else if(c.root_free_kb) parts.push(t('edge_storage_root')+' '+fmtMi(c.root_free_kb)+' free');
    if(c.flash_kb) parts.push(t('edge_storage_flash')+' '+fmtMi(c.flash_kb));
    if(c.has_mmc) parts.push('mmc');
    if(c.extroot_active) parts.push('extroot');
    if(c.has_usb_disk || (c.usb_disks&&c.usb_disks.length)) {
      const disks=c.usb_disks||[];
      if(disks.length){
        disks.forEach(d=>{
          let s='USB '+(d.name||'?');
          if(d.size_kb) s+=' '+Math.round(d.size_kb/1024/1024*10)/10+' GiB';
          if(d.fs) s+=' '+d.fs;
          if(d.mount) s+=' @ '+d.mount;
          if(d.free_kb) s+=' free '+Math.round(d.free_kb/1024)+' MiB';
          parts.push(s);
        });
      } else parts.push('usb');
    }
    const storage = parts.length ? parts.join(' · ') : '—';
    stats.innerHTML =
      '<div class="edge-stat"><b>'+t('edge_stat_mem')+'</b>'+mem+'</div>'+
      '<div class="edge-stat"><b>'+t('edge_stat_storage')+'</b>'+storage+'</div>'+
      '<div class="edge-stat"><b>'+t('edge_stat_expand')+'</b>'+(c.expand_hint?t('edge_yes'):t('edge_no'))+'</div>';
  }
  // prefill lan from facts
  const f=resp.facts||{};
  const uci=f.uci||{};
  const lip=document.getElementById('edge_lan_ip');
  if(lip && uci.lan_ip && !lip.value) lip.value=uci.lan_ip;
  const bands=c.bands||[];
  const w5=document.getElementById('edge_wifi5_row');
  if(w5) w5.classList.toggle('hidden', !(bands.includes('5g')||bands.includes('6g')));
}

function edgeRenderModules(resp){
  const list=document.getElementById('edge_mod_list');
  if(!list) return;
  list.innerHTML='';
  const adv=!!document.getElementById('edge_advanced')?.checked;
  const labels={
    agent_install:t('edge_mod_agent'), lan_baseline:t('edge_mod_lan'), wan_baseline:t('edge_mod_wan'),
    wifi_ap:t('edge_mod_wifi'), guest:t('edge_mod_guest'), vpn_client:t('edge_mod_vpn'),
    fs_expand:t('edge_mod_fs'), overlay_ext:t('edge_mod_overlay'), ssh_harden:t('edge_mod_ssh')
  };
  const hardRe=/no radios|no wan|no uplink|no expand|no external|no USB|sing-box needs|unknown module|single-NIC/i;
  (resp.plan?.steps||[]).forEach(s=>{
    const row=document.createElement('label');
    const hard = hardRe.test(s.reason||'');
    const can = s.action==='apply' || !hard || adv;
    row.className='edge-mod'+(can?'':' dim');
    const checked = s.action==='apply';
    let reason = s.reason||'';
    if(s.module==='vpn_client' && reason && !/no USB|sing-box needs/i.test(reason)) reason = t('edge_mod_vpn_help');
    row.innerHTML = '<input type="checkbox" data-mod="'+s.module+'" '+(checked?'checked':'')+' '+(can?'':'disabled')+'/>'+
      '<div><div class="edge-mod-title">'+(labels[s.module]||s.module)+'</div>'+
      '<div class="edge-mod-reason">'+s.action+(reason?(' — '+reason):'')+'</div></div>';
    list.appendChild(row);
  });
  // Opt-in format USB when disk present (PLAN-EDGE-USB-STORAGE)
  const card=resp.card||{};
  if(card.has_usb_disk || (card.usb_disks&&card.usb_disks.length)){
    const row=document.createElement('label');
    row.className='edge-mod';
    row.innerHTML = '<input type="checkbox" data-mod="usb_format"/>'+
      '<div><div class="edge-mod-title">'+(t('edge_mod_usb_format')||'Format USB')+'</div>'+
      '<div class="edge-mod-reason">'+(t('edge_mod_usb_format_help')||'Destroy data on USB for extroot')+'</div></div>';
    list.appendChild(row);
  }
  edgeSyncDetailVisibility();
}

function edgeSyncDetailVisibility(){
  const mods=edgeCollectModules();
  const map={edge_det_lan:'lan_baseline', edge_det_wan:'wan_baseline', edge_det_wifi:'wifi_ap', edge_det_guest:'guest'};
  Object.entries(map).forEach(([id,mod])=>{
    const el=document.getElementById(id);
    if(el) el.classList.toggle('hidden', !mods[mod]);
  });
  edgeIntentFromModules(mods);
  const sum=document.getElementById('edge_plan_summary');
  if(sum && window.__edgePreview){
    const steps=(window.__edgePreview.plan?.steps||[]).map(s=>{
      const en=mods[s.module];
      const act = en===false ? 'skip (user)' : (en || s.action==='apply' ? 'apply' : s.action);
      return act.padEnd(6)+' '+s.module+(s.reason?'  '+s.reason:'');
    });
    sum.textContent = steps.join('\n');
  }
}

async function edgePreview(){
  const form=document.getElementById('form-edge');
  if(!form) throw new Error('form-edge missing');
  const fd=new FormData(form);
  const router=(fd.get('router_host')||'').toString().trim();
  if(!router) throw new Error('Router (SSH) host is empty — e.g. 192.168.1.1');
  const mods=edgeCollectModules();
  const body={
    router_host:router,
    router_user:(fd.get('router_user')||'root').toString().trim()||'root',
    router_password:(fd.get('router_password')||'').toString(),
    primary_key:(fd.get('primary_key')||'').toString(),
    key_passphrase:(fd.get('key_passphrase')||'').toString(),
    preset:(fd.get('preset')||'').toString(),
    net_configure:true,
    guest_enable:true,
    vpn_enable: Object.keys(mods).length ? !!mods.vpn_client : true,
    wifi_ssid_24:(fd.get('wifi_ssid_24')||'').toString(),
    wifi_ssid_5:(fd.get('wifi_ssid_5')||'').toString(),
    advanced:!!document.getElementById('edge_advanced')?.checked,
    modules: mods
  };
  line('job edge.probe …');
  const j=await jobRun('edge.probe', body, jobLogToUI);
  if(j.status!=='succeeded'){
    throw new Error(j.error||('probe '+j.status));
  }
  let data={};
  try{
    const r=j.result||{};
    // result is {facts, plan}; card expects plan-shaped + facts
    data=Object.assign({}, r.plan||{}, {facts:r.facts, plan:r.plan, result:r});
    if(r.plan) Object.assign(data, r.plan);
  }catch(e){ data={error:String(e)}; }
  window.__edgePreview=data;
  edgeRenderCard(data);
  edgeRenderModules(data);
  const ps=document.getElementById('edge_preset');
  if(ps && data.preset && !ps.value) ps.value=data.preset;
  return data;
}

document.getElementById('btnEdgeProbe')?.addEventListener('click', async ev=>{
  const btn=ev.currentTarget; btn.disabled=true;
  try{
    await edgePreview();
    edgeSetStep(2);
  }catch(e){ alert(e.message||e); }
  finally{ btn.disabled=false; }
});
document.getElementById('btnEdgeToModules')?.addEventListener('click', ()=>edgeSetStep(3));
document.getElementById('btnEdgeToDetails')?.addEventListener('click', ()=>{
  edgeSyncDetailVisibility();
  edgeSetStep(4);
});
document.getElementById('btnEdgeBack1')?.addEventListener('click', ()=>edgeSetStep(1));
document.getElementById('btnEdgeBack2')?.addEventListener('click', ()=>edgeSetStep(2));
document.getElementById('btnEdgeBack3')?.addEventListener('click', ()=>edgeSetStep(3));
document.getElementById('btnEdgeReplan')?.addEventListener('click', async ev=>{
  const btn=ev.currentTarget; btn.disabled=true;
  try{ await edgePreview(); }catch(e){ alert(e.message||e); }
  finally{ btn.disabled=false; }
});
document.getElementById('edge_advanced')?.addEventListener('change', async ()=>{
  try{ await edgePreview(); }catch(_){}
});
document.getElementById('edge_mod_list')?.addEventListener('change', edgeSyncDetailVisibility);
document.getElementById('edge_preset')?.addEventListener('change', async ()=>{
  try{ await edgePreview(); }catch(_){}
});
document.getElementById('edge_wan_proto')?.addEventListener('change', ()=>{
  const v=document.getElementById('edge_wan_proto')?.value;
  const st=document.getElementById('edge_wan_static');
  const pp=document.getElementById('edge_wan_pppoe');
  if(st) st.hidden = v!=='static';
  if(pp) pp.hidden = v!=='pppoe';
});
document.getElementById('btnEdgeDryRun')?.addEventListener('click', async ()=>{
  try{
    const data=await edgePreview();
    edgeSyncDetailVisibility();
    alert('Plan refreshed — see summary below.');
  }catch(e){ alert(e.message||e); }
});

// Patch form-edge submit to include modules/preset/advanced
(function(){
  const form=document.getElementById('form-edge');
  if(!form) return;
  form.onsubmit=async e=>{
    e.preventDefault();
    const fd=new FormData(e.target);
    saveForm('edge',fd);
    const mods=edgeCollectModules();
    const intent=edgeIntentFromModules(mods);
    const submitBtn=e.submitter||e.target.querySelector('button[type=submit]');
    await streamPost('/v1/edge',{
      offline:!!document.getElementById('edge_offline')?.checked,
      router_host:fd.get('router_host'), router_password:fd.get('router_password')||'',
      new_root_password:(fd.get('skip_root_pass')==='on'?'':(fd.get('new_root_password')||'')),
      skip_root_pass:fd.get('skip_root_pass')==='on',
      guest_hidden:fd.get('guest_hidden')||'0',
      device_id:fd.get('device_id'), agent_arch:fd.get('agent_arch')||'auto',
      primary_host:fd.get('primary_host'), primary_key:fd.get('primary_key'),
      server_url:fd.get('server_url'), key_passphrase:fd.get('key_passphrase'),
      net_configure:intent.net_configure, guest_enable:intent.guest_enable,
      reboot:fd.get('reboot')==='on',
      preset:fd.get('preset')||'',
      advanced:!!document.getElementById('edge_advanced')?.checked,
      modules:mods,
      lan_ip:fd.get('lan_ip'), lan_mask:fd.get('lan_mask'),
      dhcp_start:fd.get('dhcp_start'), dhcp_limit:fd.get('dhcp_limit'),
      wifi_ssid_24:fd.get('wifi_ssid_24'), wifi_key_24:fd.get('wifi_key_24'),
      wifi_ssid_5:fd.get('wifi_ssid_5'), wifi_key_5:fd.get('wifi_key_5'),
      wan_proto:fd.get('wan_proto')||'dhcp',
      wan_ip:fd.get('wan_ip'), wan_mask:fd.get('wan_mask'),
      wan_gateway:fd.get('wan_gateway'), wan_dns:fd.get('wan_dns'),
      pppoe_user:fd.get('pppoe_user'), pppoe_pass:fd.get('pppoe_pass'),
      guest_ssid:fd.get('guest_ssid'), guest_pin:fd.get('guest_pin'), guest_psk:fd.get('guest_psk')
    }, submitBtn);
  };
})();

edgeSetStep(1);


/* Edge storage day-2 (PLAN-EDGE-USB-STORAGE) */
async function edgeStorageAPI(path, body){
  // path like /api/edge/storage/backup-now — via node proxy + session
  const r = await nodeFetch(path, {method:'POST', body: JSON.stringify(body||{})});
  if(!r.ok) throw new Error((r.data&&(r.data.error||r.data.message))||('HTTP '+r.status));
  return r.data;
}
function edgeStDevice(){
  return (document.getElementById('edge_st_device')?.value||document.getElementById('edge_device_id')?.value||'').trim();
}
function edgeStLog(msg){
  const el=document.getElementById('edge_st_log');
  if(el) el.textContent = (el.textContent?el.textContent+'\n':'')+msg;
}

document.getElementById('btnStList')?.addEventListener('click', async ()=>{
  try{
    const id=edgeStDevice(); if(!id) throw new Error('device_id');
    const d=await nodeFetch('/api/edge/storage/backups?device_id='+encodeURIComponent(id));
    const man=(d.backups||[]).slice(0,12).map((m,i)=> (i+1)+'. ['+(m.kind||'?')+'] '+(m.file||'')+' @ '+(m.created||'')).join('\n');
    const up=(d.uploads||[]).slice(0,12).map((u,i)=> (i+1)+'. '+(u.name||'')+' size='+(u.size||'')).join('\n');
    const el=document.getElementById('edge_st_table');
    if(el) el.textContent = 'Manifests:\n'+(man||'(empty)')+'\n\nAgent uploads:\n'+(up||'(empty)');
    edgeStLog('list ok manifests='+((d.backups||[]).length)+' uploads='+((d.uploads||[]).length));
  }catch(e){ edgeStLog(String(e.message||e)); }
});
document.getElementById('btnStBackup')?.addEventListener('click', async ()=>{
  try{
    const id=edgeStDevice(); if(!id) throw new Error('device_id');
    const d=await edgeStorageAPI('/api/edge/storage/backup-now',{device_id:id});
    edgeStLog(JSON.stringify(d));
  }catch(e){ edgeStLog(String(e.message||e)); }
});
document.getElementById('btnStRestore')?.addEventListener('click', async ()=>{
  try{
    const id=edgeStDevice(); if(!id) throw new Error('device_id');
    const d=await edgeStorageAPI('/api/edge/storage/restore',{device_id:id, format:false});
    edgeStLog(JSON.stringify(d));
  }catch(e){ edgeStLog(String(e.message||e)); }
});
document.getElementById('btnStRestoreFmt')?.addEventListener('click', async ()=>{
  if(!confirm('Format USB and restore from primary?')) return;
  try{
    const id=edgeStDevice(); if(!id) throw new Error('device_id');
    const d=await edgeStorageAPI('/api/edge/storage/restore',{device_id:id, format:true});
    edgeStLog(JSON.stringify(d));
  }catch(e){ edgeStLog(String(e.message||e)); }
});
document.getElementById('btnStMute')?.addEventListener('click', async ()=>{
  try{
    const id=edgeStDevice(); if(!id) throw new Error('device_id');
    const d=await edgeStorageAPI('/api/edge/storage/mute',{device_id:id, mute:true});
    edgeStLog(JSON.stringify(d));
  }catch(e){ edgeStLog(String(e.message||e)); }
});
document.getElementById('btnStUnmute')?.addEventListener('click', async ()=>{
  try{
    const id=edgeStDevice(); if(!id) throw new Error('device_id');
    const d=await edgeStorageAPI('/api/edge/storage/mute',{device_id:id, mute:false});
    edgeStLog(JSON.stringify(d));
  }catch(e){ edgeStLog(String(e.message||e)); }
});
document.getElementById('btnStOffline')?.addEventListener('click', async ()=>{
  try{
    const id=edgeStDevice(); if(!id) throw new Error('device_id');
    const d=await edgeStorageAPI('/api/edge/storage/offline-pack',{device_id:id});
    edgeStLog(JSON.stringify(d));
  }catch(e){ edgeStLog(String(e.message||e)); }
});

async function jobReattachOnLoad(){
  try{
    let id=null, typ=null;
    for(let i=0;i<sessionStorage.length;i++){
      const k=sessionStorage.key(i);
      if(k && k.indexOf('nd_op_job_')===0){
        const cand=sessionStorage.getItem(k);
        if(!cand) continue;
        try{
          const j=await jobGet(cand);
          if(j && (j.status==='running'||j.status==='queued')){ id=cand; typ=k; break; }
          if(j && !id) id=cand;
        }catch(e){}
      }
    }
    if(!id) return;
    const j=await jobGet(id);
    if(!j||!j.status) return;
    const logSec=document.getElementById('deployLogSec');
    if(logSec){ logSec.classList.add('deploy-log-active'); }
    resetLog(); line('re-attach job '+id+' ('+(j.type||'')+') status='+j.status);
    if(j.log) line(j.log);
    if(j.status==='running'||j.status==='queued'){
      await jobWait(id, j2=>{ if(j2.log){ resetLog(); line(j2.log);} });
    }
  }catch(e){ console.warn('job reattach', e); }
}
jobReattachOnLoad();

document.getElementById('btnJobsRefresh')?.addEventListener('click', ()=>refreshJobsPanel());
document.getElementById('btnJobsRunningOnly')?.addEventListener('click', ()=>{ jobsShowRunningOnly=!jobsShowRunningOnly; refreshJobsPanel(); });
setInterval(()=>{ try{ refreshJobsPanel(); }catch(e){} }, 4000);
refreshJobsPanel();




/* ===== Shell: hybrid sidebar / jobs drawer / statusbar / Servers cards ===== */
function openJobsDrawer(on){
  const d=document.getElementById('jobsDrawer');
  const b=document.getElementById('jobsBackdrop');
  const btn=document.getElementById('btnHeaderJobs');
  if(!d) return;
  d.classList.toggle('on', !!on);
  if(b) b.classList.toggle('on', !!on);
  if(btn) btn.classList.toggle('on', !!on);
  if(on){ try{ refreshJobsPanel(); }catch(e){} }
}
function setFleetPages(which){
  ['page-servers','page-server-detail','page-sites'].forEach(id=>{
    const el=document.getElementById(id);
    if(!el) return;
    const on = (which==='servers' && id==='page-servers')
      || (which==='detail' && id==='page-server-detail')
      || (which==='sites' && id==='page-sites');
    el.classList.toggle('hidden', !on);
  });
}
function hideLegacyMains(){
  ['main-installer','main-control','main-settings','deployLogSec','subInstaller','mainTabs'].forEach(id=>{
    const el=document.getElementById(id); if(!el) return;
    el.classList.add('hidden'); el.style.display='none';
  });
}
function showLegacyMain(id){
  const el=document.getElementById(id); if(!el) return;
  el.classList.remove('hidden'); el.style.display='';
}
function syncSideNav(){
  const mainBtn=document.querySelector('#mainTabs button.active');
  const main=mainBtn && mainBtn.dataset.main;
  let key = window.__ndNav || 'servers';
  if(main==='control') key='control';
  else if(main==='settings') key='settings';
  else if(main==='installer'){
    const tab=document.querySelector('#subInstaller button.active');
    key='installer:'+((tab && tab.dataset.tab)||'fleet');
  }
  document.querySelectorAll('#sideNav .nav-btn[data-nav], .side-foot .nav-btn[data-nav]').forEach(b=>{
    b.classList.toggle('on', b.dataset.nav===key);
  });
  const titles={
    servers:(typeof t==='function'?t('nav_servers'):'Servers'),
    sites:(typeof t==='function'?t('nav_sites'):'Sites'),
    'installer:fleet':'Fleet','installer:primary':'Primary','installer:secondary':'Secondary',
    'installer:build':'Build','installer:edge':'OpenWrt','installer:site':'MikroTik',
    'installer:nvr':'NVR','installer:creds':'Credentials',
    control:(typeof t==='function'?t('nav_control'):'Operate'),
    settings:(typeof t==='function'?t('nav_settings'):'Settings')
  };
  const pt=document.getElementById('pageTitle');
  if(pt) pt.textContent = titles[key] || key;
}
function showServersOverview(){
  window.__ndNav='servers';
  hideLegacyMains();
  setFleetPages('servers');
  // clear mainTabs active to installer-none
  document.querySelectorAll('#mainTabs button').forEach(b=>b.classList.remove('active'));
  renderServersCards();
  syncSideNav();
}

function formatUptimeUI(sec){
  if(!sec || sec<=0) return '—';
  const s=Math.floor(sec), d=Math.floor(s/86400), h=Math.floor((s%86400)/3600), m=Math.floor((s%3600)/60);
  if(d>0) return d+'d '+h+'h '+m+'m';
  if(h>0) return h+'h '+m+'m';
  return m+'m';
}
function siteTopoSVG(s){
  // pure HTML topology (no external assets)
  const wan = esc(s.wan_ip||'—');
  const rip = esc(s.router_ip||'—');
  const zones = (s.zones||[]).map(z=>{
    const on = z.enabled ? ' on' : '';
    return '<div class="zbox'+on+'"><span class="zd"></span><span class="zt">'+esc(z.title)+'</span><span class="zc">'+esc(z.cidr||'—')+'</span></div>';
  }).join('');
  return '<div class="topo">'
    +'<div class="topo-left">'
    +'<div class="topo-node"><div class="ic">☁</div><div class="lab">WAN</div><div class="ip">'+wan+'</div></div>'
    +'<div class="topo-link"></div>'
    +'<div class="topo-node"><div class="ic">▣</div><div class="lab">'+esc(s.board||s.kind||'Router')+'</div><div class="ip">'+rip+'</div></div>'
    +'</div>'
    +'<div class="topo-zones">'+zones+'</div>'
    +'</div>';
}
async function renderSitesCards(){
  const grid=document.getElementById('sitesGrid');
  if(!grid) return;
  grid.innerHTML='<span class="note">…</span>';
  let sites=[];
  try{
    const j=await nodeFetch('/api/edge/sites');
    sites = (j && j.sites) || [];
  }catch(e){
    grid.innerHTML='<span class="note">'+(typeof t==='function'?t('sites_offline'):'Need tunnel + session')+'</span>';
    // still show add-site affordance via empty state button area
    return;
  }
  if(!sites.length){
    grid.innerHTML='<span class="note">'+(typeof t==='function'?t('sites_empty'):'No sites')+'</span>';
    return;
  }
  grid.innerHTML='';
  sites.forEach(s=>{
    const card=document.createElement('div');
    card.className='site-card';
    const st=s.status||'draft';
    const dot = st==='online'?' on':(st==='offline'?' warn':'');
    const kindLabel = [st==='online'?'Online':st, s.board||s.kind].filter(Boolean).join(' | ');
    card.innerHTML =
      '<div class="head"><span class="dot'+dot+'"></span><span>Site · '+esc(s.name||s.device_id)+'</span></div>'
      +'<div class="sub">'+esc(kindLabel)+'</div>'
      + siteTopoSVG(s)
      +'<div class="site-foot">'
      +'<div class="sf"><div class="sk">Devices</div><div class="sv">'+(s.devices!=null&&s.devices>0?s.devices:'—')+'</div></div>'
      +'<div class="sf"><div class="sk">Guests</div><div class="sv">'+(s.guests!=null&&s.guests>0?s.guests:'—')+'</div></div>'
      +'<div class="sf"><div class="sk">Uptime</div><div class="sv">'+(s.uptime_fmt||formatUptimeUI(s.uptime_sec)||'—')+'</div></div>'
      +'<div class="sf"><div class="sk">Latency</div><div class="sv'+(s.latency_ms>0?' ok':'')+'">'+(s.latency_ms>0?(Math.round(s.latency_ms)+' ms'):'—')+'</div></div>'
      +'</div>';
    card.onclick=()=>{
      // open edge deploy with device context later; for now edge wizard
      navigateShell('installer:edge');
    };
    grid.appendChild(card);
  });
}

function showSitesOverview(){
  window.__ndNav='sites';
  hideLegacyMains();
  setFleetPages('sites');
  document.querySelectorAll('#mainTabs button').forEach(b=>b.classList.remove('active'));
  try{ renderSitesCards(); }catch(e){}
  syncSideNav();
}
function showServerDetail(role){
  window.__ndNav='servers';
  hideLegacyMains();
  setFleetPages('detail');
  const s=settings();
  const map={
    primary:{name:'nd-primary', role:'primary', host:s.primary_host||'—', key:s.key||'~/.ssh/netductor_primary'},
    secondary:{name:'nd-secondary', role:'secondary', host:s.secondary_host||'—', key:s.key||''},
    build:{name:'build-host', role:'build', host:s.build_host||'—', key:s.key||''}
  };
  const m=map[role]||map.primary;
  const dn=document.getElementById('srvDetailName');
  if(dn) dn.innerHTML=m.name+' <span class="badge">'+m.role+'</span>';
  const meta=document.getElementById('srvDetailMeta');
  if(meta) meta.textContent=m.host+' · key '+m.key;
  const metrics=document.getElementById('srvDetailMetrics');
  if(metrics){
    metrics.innerHTML='<div class="metric"><div class="k">Host</div><div class="v" style="font-size:.9rem">'+esc(m.host)+'</div><div class="s">from Settings</div></div>'
      +'<div class="metric"><div class="k">Role</div><div class="v" style="font-size:.9rem">'+m.role+'</div><div class="s">control plane</div></div>'
      +'<div class="metric"><div class="k">Metrics</div><div class="v" style="font-size:.9rem">—</div><div class="s">snapshot via Operate</div></div>'
      +'<div class="metric"><div class="k">API</div><div class="v" style="font-size:.9rem">'+(s.api_port||'8787')+'</div><div class="s">tunnel</div></div>';
  }
  // topology under metrics
  let topoHost=document.getElementById('srvDetailTopo');
  if(!topoHost){
    topoHost=document.createElement('div');
    topoHost.id='srvDetailTopo';
    if(metrics && metrics.parentNode) metrics.parentNode.insertBefore(topoHost, metrics.nextSibling);
  }
  topoHost.innerHTML=fleetTopoHTML(m.role, m.host);
  const act=document.getElementById('srvDetailActions');
  if(act){
    act.innerHTML='';
    const add=(label, nav)=>{
      const b=document.createElement('button');
      b.type='button'; b.textContent=label;
      b.className='primary';
      b.onclick=()=>navigateShell(nav);
      act.appendChild(b);
    };
    add(typeof t==='function'?t('nav_control'):'Operate', 'control');
    if(role==='primary') add('Deploy primary', 'installer:primary');
    if(role==='secondary') add('Deploy secondary', 'installer:secondary');
    if(role==='build') add('Deploy build', 'installer:build');
    add(typeof t==='function'?t('nav_settings'):'Settings', 'settings');
  }
  const pt=document.getElementById('pageTitle');
  if(pt) pt.textContent=(typeof t==='function'?t('nav_servers'):'Servers')+' / '+m.name;
}
function fleetTopoHTML(role, host){
  // Schematic for control-plane nodes (same visual language as site topology).
  const h=esc(host||'—');
  if(role==='primary'){
    return '<div class="fleet-topo">'
      +'<div class="topo-node"><div class="ic">▣</div><div class="lab">Primary</div><div class="ip">'+h+'</div></div>'
      +'<div class="topo-link"></div>'
      +'<div class="topo-mid">'
      +'<div class="zbox on"><span class="zd"></span><span class="zt">SP</span><span class="zc">10.87.10/30</span></div>'
      +'<div class="zbox on"><span class="zd"></span><span class="zt">PS</span><span class="zc">10.87.11/30</span></div>'
      +'<div class="zbox on"><span class="zd"></span><span class="zt">mTLS</span><span class="zc">:8789</span></div>'
      +'</div>'
      +'<div class="topo-link"></div>'
      +'<div class="topo-node"><div class="ic">◇</div><div class="lab">Secondary</div><div class="ip">peer</div></div>'
      +'</div>';
  }
  if(role==='secondary'){
    return '<div class="fleet-topo">'
      +'<div class="topo-node"><div class="ic">◇</div><div class="lab">Secondary</div><div class="ip">'+h+'</div></div>'
      +'<div class="topo-link"></div>'
      +'<div class="topo-mid">'
      +'<div class="zbox on"><span class="zd"></span><span class="zt">uplink</span><span class="zc">VLESS :443</span></div>'
      +'<div class="zbox on"><span class="zd"></span><span class="zt">agent</span><span class="zc">hb → primary</span></div>'
      +'</div>'
      +'<div class="topo-link"></div>'
      +'<div class="topo-node"><div class="ic">▣</div><div class="lab">Primary</div><div class="ip">control</div></div>'
      +'</div>';
  }
  // build
  return '<div class="fleet-topo">'
    +'<div class="topo-node"><div class="ic">▣</div><div class="lab">Primary</div><div class="ip">control</div></div>'
    +'<div class="topo-link"></div>'
    +'<div class="topo-mid">'
    +'<div class="zbox"><span class="zd"></span><span class="zt">PB/BP</span><span class="zc">10.87.12–13</span></div>'
    +'<div class="zbox"><span class="zd"></span><span class="zt">jobs</span><span class="zc">:8789</span></div>'
    +'</div>'
    +'<div class="topo-link"></div>'
    +'<div class="topo-node"><div class="ic">⬡</div><div class="lab">Build</div><div class="ip">'+h+'</div></div>'
    +'</div>';
}
function renderServersCards(){
  const grid=document.getElementById('serversGrid');
  if(!grid) return;
  const s=settings();
  const cards=[
    {id:'primary', name:'nd-primary', role:'primary', host:s.primary_host||'—'},
    {id:'secondary', name:'nd-secondary', role:'secondary', host:s.secondary_host||'—'},
    {id:'build', name:'build-host', role:'build', host:s.build_host||'(optional)'}
  ];
  const chips=document.getElementById('serversChips');
  if(chips){
    const n=cards.filter(c=>c.host && c.host!=='—' && c.host!=='(optional)').length;
    chips.innerHTML='<span class="badge">'+(n)+' configured</span>';
  }
  grid.innerHTML='';
  // overview topology strip (fleet map)
  const map=document.createElement('div');
  map.className='card';
  map.style.gridColumn='1 / -1';
  map.innerHTML='<h3 style="margin:0 0 .35rem;font-size:.75rem;text-transform:uppercase;letter-spacing:.06em;color:var(--faint)">'
    +(uiLang==='ru'?'Каналы control-plane':'Control-plane paths')+'</h3>'
    +'<div class="fleet-topo">'
    +'<div class="topo-node"><div class="ic">▣</div><div class="lab">Primary</div><div class="ip">'+esc(s.primary_host||'—')+'</div></div>'
    +'<div class="topo-link"></div>'
    +'<div class="topo-mid">'
    +'<div class="zbox on"><span class="zd"></span><span class="zt">SP</span><span class="zc">10.87.10/30</span></div>'
    +'<div class="zbox on"><span class="zd"></span><span class="zt">PS</span><span class="zc">10.87.11/30</span></div>'
    +'<div class="zbox on"><span class="zd"></span><span class="zt">mTLS</span><span class="zc">:8789</span></div>'
    +'</div>'
    +'<div class="topo-link"></div>'
    +'<div class="topo-node"><div class="ic">◇</div><div class="lab">Secondary</div><div class="ip">'+esc(s.secondary_host||'—')+'</div></div>'
    +(s.build_host?('<div class="topo-link"></div><div class="topo-node"><div class="ic">⬡</div><div class="lab">Build</div><div class="ip">'+esc(s.build_host)+'</div></div>'):'')
    +'</div>';
  grid.appendChild(map);
  cards.forEach(c=>{
    const el=document.createElement('div');
    el.className='srv-card';
    const ok=c.host && c.host!=='—' && c.host!=='(optional)';
    el.innerHTML='<div class="row1"><span class="name">'+esc(c.name)+'</span><span class="badge">'+c.role+'</span></div>'
      +'<div class="meta">'+esc(c.host)+'</div>'
      + fleetTopoHTML(c.role, c.host)
      +'<div class="kv"><span>Status</span><b>'+(ok?'configured':'not set')+'</b></div>';
    el.onclick=()=>showServerDetail(c.id);
    grid.appendChild(el);
  });
}
function navigateShell(nav){
  if(!nav) return;
  window.__ndNav=nav;
  if(nav==='servers'){ showServersOverview(); return; }
  if(nav==='sites'){ showSitesOverview(); return; }
  setFleetPages(''); // hide all fleet pages
  document.querySelectorAll('.fleet-page').forEach(p=>p.classList.add('hidden'));
  if(nav==='control'){ showMain('control'); }
  else if(nav==='settings'){ showMain('settings'); }
  else if(nav.startsWith('installer:')){
    const tab=nav.split(':')[1]||'fleet';
    showMain('installer');
    const btn=document.querySelector('#subInstaller button[data-tab="'+tab+'"]');
    if(btn) btn.click();
  }
  syncSideNav();
}
function initShell(){
  document.getElementById('btnCollapse')?.addEventListener('click', ()=>{
    document.body.classList.toggle('collapsed');
    try{ localStorage.setItem('nd_op_collapsed', document.body.classList.contains('collapsed')?'1':'0'); }catch(e){}
  });
  try{ if(localStorage.getItem('nd_op_collapsed')==='1') document.body.classList.add('collapsed'); }catch(e){}
  document.querySelectorAll('[data-nav]').forEach(b=>{
    b.addEventListener('click', ()=>navigateShell(b.dataset.nav));
  });
  document.getElementById('btnHeaderJobs')?.addEventListener('click', ()=>{
    const d=document.getElementById('jobsDrawer');
    openJobsDrawer(!(d && d.classList.contains('on')));
  });
  document.getElementById('btnCloseJobs')?.addEventListener('click', ()=>openJobsDrawer(false));
  document.getElementById('jobsBackdrop')?.addEventListener('click', ()=>openJobsDrawer(false));
  document.getElementById('sbJobsMini')?.addEventListener('click', ()=>openJobsDrawer(true));
  document.getElementById('sbTheme')?.addEventListener('click', ()=>{
    document.body.classList.toggle('light');
    try{ localStorage.setItem('nd_op_light', document.body.classList.contains('light')?'1':'0'); }catch(e){}
  });
  try{ if(localStorage.getItem('nd_op_light')==='1') document.body.classList.add('light'); }catch(e){}
  function tick(){
    const el=document.getElementById('sbClock');
    if(!el) return;
    const d=new Date(), p=n=>String(n).padStart(2,'0');
    el.textContent=p(d.getHours())+':'+p(d.getMinutes())+':'+p(d.getSeconds());
  }
  tick(); setInterval(tick, 1000);
  const badge=document.getElementById('jobsRunningCount');
  if(badge){
    const mo=new MutationObserver(()=>{
      const n=parseInt(badge.textContent,10)||0;
      const sb=document.getElementById('sbJobsN');
      if(sb) sb.textContent=String(n);
      badge.classList.toggle('hide', n<=0);
    });
    mo.observe(badge, {characterData:true, childList:true, subtree:true});
  }
  document.getElementById('btnServersBack')?.addEventListener('click', ()=>showServersOverview());
  document.getElementById('btnAddSite')?.addEventListener('click', ()=>navigateShell('installer:edge'));
  document.getElementById('btnSitesRefresh')?.addEventListener('click', ()=>renderSitesCards());
  document.querySelectorAll('[data-site-go]').forEach(c=>{
    c.addEventListener('click', ()=>navigateShell('installer:'+c.getAttribute('data-site-go')));
  });
  document.querySelectorAll('#subInstaller button').forEach(btn=>{
    btn.addEventListener('click', ()=>setTimeout(syncSideNav, 0));
  });
  async function pollOpHealth(){
    try{
      const res=await fetch('/v1/health',{headers:{'X-Netductor-Token':ND_TOKEN}});
      const ok=res.ok;
      const d=document.getElementById('sbApi');
      if(d){ d.className='sb-dot'+(ok?'':' err'); }
      const ver=document.getElementById('sbVer');
      if(ver && ok){
        try{
          const m=await fetch('/v1/meta',{headers:{'X-Netductor-Token':ND_TOKEN}});
          const j=await m.json();
          if(j.version) ver.textContent='op '+j.version;
          else if(j.op_version) ver.textContent='op '+j.op_version;
        }catch(e){}
      }
    }catch(e){
      const d=document.getElementById('sbApi');
      if(d) d.className='sb-dot err';
    }
  }
  pollOpHealth(); setInterval(pollOpHealth, 15000);
  // default home = Servers cards
  showServersOverview();
  setTimeout(showServersOverview, 0);
  setTimeout(showServersOverview, 50);
}
function highlightSkinNav(){ try{ syncSideNav(); }catch(e){} }

function bootOp(){
  try{ initShell(); }catch(e){ console.warn(e); }
  try{ initResultDock(); }catch(e){ console.warn('result dock', e); }
}
if(document.readyState==='loading') document.addEventListener('DOMContentLoaded', bootOp);
else bootOp();
