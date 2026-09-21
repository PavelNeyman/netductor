(() => {
  const $ = (s) => document.querySelector(s);
  const state = { token: localStorage.getItem('nd_token') || '', lang: localStorage.getItem('nd_lang') || 'en' };
  const i18n = {
      app_title: 'Netductor',
      opt_auto: 'auto',
      opt_mikrotik: 'mikrotik',
      opt_secondary: 'secondary',
      backup_hint_long: 'Cross-VPS SCP of encrypted /etc/netductor (few MB). peer-set then Run backup.',
      status_hint: 'Core services + nodes + VPN (TG parity)',
      sites_hint: 'One site = ROS (routing only) + RPi OpenWrt (VPN edge). Managed as pair.',
      th_host: 'Host',
      th_wan: 'WAN',
      nodes_rename_hint: 'Format: nd-<role>-<marker>. Roles: core, edge, lab. Bidirectional on heartbeat.',
      vpn_client: 'VPN client',
      advanced_json: 'Advanced: raw JSON',
      sni_hint: 'Reality handshake names for carrier WL experiments (Yota etc.)',
      unit: 'Unit',
      mtls_hint_long: 'Agent plane client certificates (expiry / revoke). After rotate, device auto-pulls mtls_refresh.',
      socks_opt: 'socks (no TUN, recommended)',
      tun_opt: 'tun (full tunnel)',
    en: {
      mtls_title: 'mTLS certs',
      mtls_refresh: 'Refresh certs',
      mtls_hint: 'Agent plane client certificates. After rotate, device pulls mtls_refresh.',
      secondary_tab: 'Secondary',
      login_title: 'Operator sign-in',
      login_hint: 'Token: netductor vpn session 72',
      sign_in: 'Sign in',
      logout: 'Logout',
      session_token: 'Session token',
      alerts: 'Alerts',
      refresh_alerts: 'Refresh alerts',
      tab_overview: 'Overview',
      tab_vpn: 'VPN',
      tab_nodes: 'Nodes',
      tab_routers: 'Routers',
      tab_sites: 'Sites',
      tab_templates: 'Templates',
      tab_metrics: 'Metrics',
      tab_probes: 'Probes',
      tab_addons: 'Addons',
      tab_sni: 'SNI',
      tab_secondary: 'Secondary',
      tab_backup: 'Backup',
      tab_git: 'Git',
      git_hint: 'Bare repos · SSH push',
      git_init: 'Init',
      git_log: 'Log',
      git_show: 'Show / diff',
      git_show_btn: 'Show',
      git_pipelines: 'Pipelines',
      git_run: 'Run',
      git_delete: 'Delete',
      tab_audit: 'Audit',
      tab_sessions: 'Sessions',
      tab_status: 'Status',
      tab_sshhosts: 'SSH-хосты',
      th_id: 'ID',
      th_hostname: 'Hostname',
      th_role: 'Роль',
      th_kind: 'Тип',
      th_ip: 'IP',
      th_desired: 'Desired',
      lan_ip: 'LAN IP',
      lan_mask: 'Маска LAN',
      wifi_ssid: 'SSID',
      wifi_pass: 'Пароль Wi‑Fi',
      network: 'Сеть',
      wifi: 'Wi‑Fi',
      mtls_title: 'mTLS сертификаты',
      mtls_refresh: 'Обновить',
      mtls_hint: 'После rotate устройство само тянет mtls_refresh.',
      secondary_tab: 'Secondary',
      risk_redirect: 'Redirect по умолчанию loopback; публичный :80 опционален.',

      th_id: 'ID',
      th_hostname: 'Hostname',
      th_role: 'Role',
      th_kind: 'Kind',
      th_ip: 'IP',
      th_desired: 'Desired',
      lan_ip: 'LAN IP',
      lan_mask: 'LAN mask',
      wifi_ssid: 'SSID',
      wifi_pass: 'Wi‑Fi password',
      network: 'Network',
      wifi: 'Wi‑Fi',
      mtls_title: 'mTLS certs',
      mtls_refresh: 'Refresh certs',
      mtls_hint: 'After rotate, device pulls mtls_refresh automatically.',
      secondary_tab: 'Secondary',
      risk_redirect: 'Redirect default is loopback; public :80 is optional.',

      overview_title: 'Overview',
      collector_stale: 'Collector stale',
      add: 'Add',
      refresh: 'Refresh',
      th_name: 'Name',
      th_status: 'Status',
      th_note: 'Note',
      user_detail: 'User',
      copy_sub: 'Copy subscription',
      copy_vless: 'VLESS (primary)',
      copy_hy2: 'HY2 (optional)',
      rename: 'Rename',
      sites_title: 'Sites (MikroTik + RPi)',
      save_site: 'Save site',
      mt_rsc: 'MT RSC',
      sites_list: 'Sites',
      edge_recovery_title: 'Edge recovery / register',
      edge_recovery_hint: 'LAN page on router: http://<router>:7879/netductor-recovery. Control plane: mTLS :8789 (client cert).',
      issue_recovery: 'Issue recovery code',
      register_pending: 'Register pending',
      set_site: 'Set site',
      rename_node: 'Rename node',
      template: 'Template',
      bind_device: 'Bind to device',
      nodes_title: 'Nodes',
      routers_title: 'Routers / edge',
      templates_title: 'Templates',
      metrics_title: 'Metrics',
      probes_title: 'Probes',
      sni_title: 'SNI',
      relay_title: 'Secondary (RU)',
      relay_hint: 'Mobile path: phone → RU secondary → primary.',
      export_bundle: 'Export bundle (ya.ru)',
      ru_exit_on: 'RU exit ON',
      ru_exit_off: 'RU exit OFF',
      oneliner_ru: 'One-liner for RU VPS',
      after_join_ru: 'After join on RU: netductor secondary links',
      addons_title: 'Addons',
      lampac: 'Lampac',
      lampac_hint: 'Bound to loopback only (VPN/SSH). Open UI via tunnel.',
      open_ui: 'Open UI',
      lampac_admin: 'Lampac Admin',
      node_ops: 'Node ops',
      node_ops_hint: 'Same as Telegram: journal + restart service',
      journal: 'Journal',
      restart_svc: 'Restart service',
      relay_routing: 'Secondary routing',
      relay_routing_hint: 'RU domains → direct on secondary · other → primary. See docs/RELAY-ROUTING.md',
      sync_relay: 'Sync secondary config',
      cpu: 'CPU',
      ram: 'RAM',
      disk: 'Disk',
      load: 'Load',
      health: 'Health',
      cpu_ram: 'CPU / RAM',
      version: 'Version',
      ping: 'Ping',
      chromium: 'Chromium',
      ssh_hosts_title: 'SSH known hosts (TOFU)',
      ssh_hosts_hint: 'After MT/VPS reinstall: Forget key, then reconnect once from trusted LAN.',
      audit_title: 'Audit log',
      sessions_title: 'Sessions',
      backup_title: 'Backup / peer',
      status_title: 'Status',
      clear_mt: 'Clear MikroTik',
      clear_secondary: 'Clear Secondary',
      forget: 'Forget',
      revoke_all: 'Revoke all',
      refresh_vpn_links: 'Refresh VPN links',
      set_peer: 'Set peer',
      run_backup: 'Run backup now',
      export_devices: 'Export devices JSON',
      upgrade: 'Upgrade',
      reboot: 'Reboot',
      sync_relays: 'Sync secondaries',
      set_hostname: 'Set desired hostname',
      ensure_default: 'Ensure default',
      save_template: 'Save template',
      load_form: 'Load into form',
      bind: 'Bind',
      load_json: 'Load JSON',
      save_json: 'Save JSON',
      apply_sni: 'Apply live SNI',
      sni_presets: 'SNI presets',
    },
    ru: {
      app_title: 'Netductor',
      opt_auto: 'авто',
      opt_mikrotik: 'mikrotik',
      opt_secondary: 'secondary',
      backup_hint_long: 'SCP между VPS шифрованного /etc/netductor. peer-set, затем Run backup.',
      status_hint: 'Сервисы + ноды + VPN (как в TG)',
      sites_hint: 'Один сайт = ROS (маршруты) + RPi OpenWrt (VPN edge).',
      th_host: 'Хост',
      th_wan: 'WAN',
      nodes_rename_hint: 'Формат: nd-<role>-<marker>. Роли: core, edge, lab. Подтверждение на heartbeat.',
      vpn_client: 'VPN-клиент',
      advanced_json: 'Расширенно: raw JSON',
      sni_hint: 'Reality SNI для экспериментов с белыми списками (Yota и др.)',
      unit: 'Юнит',
      mtls_hint_long: 'Клиентские сертификаты agent plane. После rotate устройство тянет mtls_refresh.',
      socks_opt: 'socks (без TUN, рекомендуется)',
      tun_opt: 'tun (полный тоннель)',
      network: 'Сеть',
      wifi: 'Wi‑Fi',
      lan_ip: 'LAN IP',
      lan_mask: 'Маска LAN',
      wifi_ssid: 'SSID',
      wifi_pass: 'Пароль Wi‑Fi',
      th_id: 'ID',
      th_hostname: 'Hostname',
      th_role: 'Роль',
      th_kind: 'Тип',
      th_ip: 'IP',
      th_desired: 'Desired',
      mtls_title: 'mTLS сертификаты',
      mtls_refresh: 'Обновить',
      mtls_hint: 'После rotate устройство тянет mtls_refresh.',
      mtls_title: 'mTLS сертификаты',
      mtls_refresh: 'Обновить',
      mtls_hint: 'Клиентские сертификаты agent plane. После rotate устройство тянет mtls_refresh.',
      secondary_tab: 'Secondary',
      login_title: 'Вход оператора',
      login_hint: 'Токен: netductor vpn session 72',
      sign_in: 'Войти',
      th_id: 'ID',
      th_hostname: 'Hostname',
      th_role: 'Роль',
      th_kind: 'Тип',
      th_ip: 'IP',
      th_desired: 'Desired',
      lan_ip: 'LAN IP',
      lan_mask: 'Маска LAN',
      wifi_ssid: 'SSID',
      wifi_pass: 'Пароль Wi‑Fi',
      network: 'Сеть',
      wifi: 'Wi‑Fi',
      mtls_title: 'mTLS сертификаты',
      mtls_refresh: 'Обновить',
      mtls_hint: 'После rotate устройство само тянет mtls_refresh.',
      secondary_tab: 'Secondary',
      risk_redirect: 'Redirect по умолчанию loopback; публичный :80 опционален.',

      logout: 'Выйти',
      session_token: 'Токен сессии',
      alerts: 'Алерты',
      refresh_alerts: 'Обновить алерты',
      tab_overview: 'Обзор',
      tab_vpn: 'VPN',
      tab_nodes: 'Ноды',
      tab_routers: 'Роутеры',
      tab_sites: 'Сайты',
      tab_templates: 'Шаблоны',
      tab_metrics: 'Метрики',
      tab_probes: 'Пробы',
      tab_addons: 'Аддоны',
      tab_sni: 'SNI',
      tab_secondary: 'Secondary',
      tab_backup: 'Бэкап',
      tab_audit: 'Аудит',
      tab_sessions: 'Сессии',
      tab_status: 'Статус',
      tab_sshhosts: 'SSH hosts',
      overview_title: 'Обзор',
      collector_stale: 'Collector устарел',
      add: 'Добавить',
      refresh: 'Обновить',
      th_name: 'Имя',
      th_status: 'Статус',
      th_note: 'Заметка',
      user_detail: 'Пользователь',
      copy_sub: 'Копировать подписку',
      copy_vless: 'VLESS (primary)',
      copy_hy2: 'HY2 (опционально)',
      rename: 'Переименовать',
      sites_title: 'Сайты (MikroTik + RPi)',
      save_site: 'Сохранить сайт',
      mt_rsc: 'MT RSC',
      sites_list: 'Сайты',
      edge_recovery_title: 'Edge recovery / регистрация',
      edge_recovery_hint: 'LAN-страница на роутере: http://<router>:7879/netductor-recovery. Control plane: mTLS :8789 (клиентский сертификат).',
      issue_recovery: 'Выдать recovery-код',
      register_pending: 'Зарегистрировать pending',
      set_site: 'Привязать сайт',
      rename_node: 'Переименовать ноду',
      template: 'Шаблон',
      bind_device: 'Привязать к устройству',
      nodes_title: 'Ноды',
      routers_title: 'Роутеры / edge',
      templates_title: 'Шаблоны',
      metrics_title: 'Метрики',
      probes_title: 'Пробы',
      sni_title: 'SNI',
      relay_title: 'Secondary (RU)',
      relay_hint: 'Путь телефона: phone → RU secondary → primary.',
      export_bundle: 'Экспорт bundle (ya.ru)',
      ru_exit_on: 'RU exit ВКЛ',
      ru_exit_off: 'RU exit ВЫКЛ',
      oneliner_ru: 'One-liner для RU VPS',
      after_join_ru: 'После join на RU: netductor secondary links',
      addons_title: 'Аддоны',
      lampac: 'Lampac',
      lampac_hint: 'Только loopback (VPN/SSH). UI через туннель.',
      open_ui: 'Открыть UI',
      lampac_admin: 'Lampac Admin',
      node_ops: 'Операции с нодой',
      node_ops_hint: 'Как в Telegram: journal + restart service',
      journal: 'Journal',
      restart_svc: 'Перезапустить сервис',
      relay_routing: 'Маршрутизация secondary',
      relay_routing_hint: 'RU-домены → direct на secondary · остальное → primary. См. docs/RELAY-ROUTING.md',
      sync_relay: 'Синхронизировать secondary',
      cpu: 'CPU',
      ram: 'RAM',
      disk: 'Диск',
      load: 'Load',
      health: 'Health',
      cpu_ram: 'CPU / RAM',
      version: 'Версия',
      ping: 'Ping',
      chromium: 'Chromium',
      ssh_hosts_title: 'SSH known hosts (TOFU)',
      ssh_hosts_hint: 'После переустановки MT/VPS: Forget key, затем один reconnect из доверенной сети.',
      audit_title: 'Журнал аудита',
      sessions_title: 'Сессии',
      backup_title: 'Бэкап / peer',
      status_title: 'Статус',
      clear_mt: 'Очистить MikroTik',
      clear_secondary: 'Очистить Secondary',
      forget: 'Забыть',
      revoke_all: 'Отозвать все',
      refresh_vpn_links: 'Обновить VPN links',
      set_peer: 'Задать peer',
      run_backup: 'Сделать бэкап',
      export_devices: 'Экспорт devices JSON',
      upgrade: 'Upgrade',
      reboot: 'Reboot',
      sync_relays: 'Синхронизировать secondary',
      set_hostname: 'Задать hostname',
      ensure_default: 'Ensure default',
      save_template: 'Сохранить шаблон',
      load_form: 'Загрузить в форму',
      bind: 'Привязать',
      load_json: 'Load JSON',
      save_json: 'Save JSON',
      apply_sni: 'Применить SNI',
      sni_presets: 'Пресеты SNI',
    },
  };
  function t(k) { return (i18n[state.lang] || i18n.en)[k] || k; }
  function applyI18n() {
    document.querySelectorAll('[data-i18n]').forEach((el) => {
      const v = t(el.dataset.i18n);
      if (el.tagName === 'TITLE') { document.title = v; return; }
      el.textContent = v;
    });
  }
  function toast(msg) {
    const el = $('#toast'); el.textContent = msg; el.classList.remove('hide');
    setTimeout(() => el.classList.add('hide'), 2500);
  }
  async function api(path, opts = {}) {
    const headers = Object.assign({ 'Content-Type': 'application/json' }, opts.headers || {});
    if (state.token) headers['Authorization'] = 'Bearer ' + state.token;
    const r = await fetch(path, { ...opts, headers });
    if (r.status === 401) { logout(); throw new Error('unauthorized'); }
    return r;
  }
  function logout() {
    state.token = ''; localStorage.removeItem('nd_token');
    $('#dash').classList.add('hide'); $('#login').classList.remove('hide');
    const blp=$('#btn-lp-refresh'); if(blp) blp.onclick=()=>refreshAddons();
  $('#btn-logout').classList.add('hide');
  }
  function showDash() {
    $('#login').classList.add('hide'); $('#dash').classList.remove('hide');
    $('#btn-logout').classList.remove('hide');
    refreshAll();
  }
  function tab(name) {
    document.querySelectorAll('.tab').forEach((b) => b.classList.toggle('active', b.dataset.tab === name));
    document.querySelectorAll('.tab-panel').forEach((p) => p.classList.toggle('hide', p.id !== 'tab-' + name));
    if (name === 'addons') refreshAddons();
    if (name === 'relay') refreshRelay();
    if (name === 'git') refreshGit();
  }
  async function refreshNodes() {
    const data = await (await api('/api/nodes')).json();
    const list = data.nodes || [];
    const tb = $('#nodes-table tbody'); if (!tb) return;
    tb.innerHTML = '';
    list.forEach((n) => {
      const tr = document.createElement('tr');
      tr.innerHTML = `<td>${n.id||''}</td><td>${n.hostname||''}</td><td>${n.role||''}</td><td>${n.kind||''}</td><td>${n.public_ip||''}</td><td>${n.desired_hostname||''}</td>
        <td><button class="ghost btn-nren" data-id="${n.id}" data-hn="${n.hostname||''}">Rename</button></td>`;
      tb.appendChild(tr);
    });
    tb.querySelectorAll('.btn-nren').forEach((b) => b.onclick = () => {
      $('#node-id').value = b.dataset.id;
      $('#node-hn').value = b.dataset.hn || '';
    });
  }
  async function refreshRelay() {
    try {
      const r = await api('/api/relay/export?sni=ya.ru');
      const b = await r.json();
      $('#relay-bundle').textContent = JSON.stringify(b, null, 2);
      try {
        const st = await (await api('/api/relay/status')).json();
        const ex = await (await api('/api/relay/exit')).json();
        if ($('#relay-exit-st')) $('#relay-exit-st').textContent = 'exit_enabled=' + ex.exit_enabled + ' · devices=' + (st.devices||[]).length;
      } catch(e) {}
      const enc = btoa(unescape(encodeURIComponent(JSON.stringify(b))));
      const cmd = 'wget -qO /usr/local/bin/netductor https://github.com/PavelNeyman/netductor/releases/download/v0.8.12/netductor-linux-amd64 && chmod 755 /usr/local/bin/netductor && echo '+enc+' | base64 -d > /root/bundle.json && netductor secondary join /root/bundle.json';
      $('#relay-oneline').textContent = cmd;
    } catch (e) {
      $('#relay-bundle').textContent = 'error: ' + e;
    }
  }
  async function refreshAddons() {
    try {
      const lp = await (await api('/api/addons/lampac')).json();
      const st = !lp.installed ? 'not installed' : (lp.running ? 'running' : 'stopped');
      $('#lp-status').textContent = st + (lp.image ? ' · ' + lp.image : '');
      $('#lp-health').textContent = lp.healthy ? 'healthy' : (lp.running ? 'unhealthy' : '—');
      $('#lp-res').textContent = (lp.cpu || '—') + ' / ' + (lp.mem || '—');
      $('#lp-ver').textContent = lp.version_hash || '—';
      $('#lp-ping').textContent = lp.ping_ok ? 'ok' : 'fail';
      $('#lp-chr').textContent = lp.chromium_ok ? 'ok' : 'fail';
      if (lp.ui_url) $('#lp-ui').href = lp.ui_url;
      if (lp.admin_url) $('#lp-admin').href = lp.admin_url;
    } catch (e) {
      $('#lp-status').textContent = 'error';
    }
  }

  async function refreshOverview() {
    try {
      const st = await (await api('/api/status')).json();
      $('#host-line').textContent = st.hostname || st.host || '';
      const m = await (await api('/api/metrics')).json().catch(() => ({}));
      $('#m-cpu').textContent = m.cpu_pct != null ? m.cpu_pct + '%' : (m.cpu || '—');
      $('#m-ram').textContent = m.mem_pct != null ? m.mem_pct + '%' : '—';
      $('#m-disk').textContent = m.disk_pct != null ? m.disk_pct + '%' : '—';
      $('#m-load').textContent = (m.load && (m.load['1'] || m.load[0])) || '—';
      $('#m-rx').textContent = m.net_rx || m.rx || '—';
      $('#m-tx').textContent = m.net_tx || m.tx || '—';
      const pr = await (await api('/api/probes')).json().catch(() => []);
      const strip = $('#probe-strip'); strip.innerHTML = '';
      (Array.isArray(pr) ? pr : (pr.probes || [])).forEach((p) => {
        const s = document.createElement('span');
        s.className = 'pill ' + (p.ok ? 'ok' : 'bad');
        s.textContent = (p.name || p.id || 'probe') + (p.ok ? ' ✓' : ' ✗');
        strip.appendChild(s);
      });
    } catch (e) { console.warn(e); }
  }
  async function refreshUsers() {
    const r = await api('/vpn/users');
    const data = await r.json();
    const users = data.users || data || [];
    const tb = $('#users-table tbody'); tb.innerHTML = '';
    users.forEach((u) => {
      const tr = document.createElement('tr');
      tr.innerHTML = `<td>${u.name}</td><td>${u.enabled ? 'on' : 'off'}</td><td>${u.note || ''}</td>
        <td><button class="ghost btn-open" data-name="${u.name}">Open</button></td>`;
      tb.appendChild(tr);
    });
    tb.querySelectorAll('.btn-open').forEach((b) => b.onclick = () => openUser(b.dataset.name));
  }
  async function openUser(name) {
    $('#user-detail').classList.remove('hide');
    $('#detail-title').textContent = name;
    const r = await api('/vpn/users/' + name + '/subscription');
    const text = await r.text();
    $('#detail-sub').textContent = text;
    const img = $('#detail-qr');
    img.classList.add('hide');
    try {
      const qr = await api('/vpn/users/' + name + '/qr');
      if (qr.ok) {
        img.src = URL.createObjectURL(await qr.blob());
        img.classList.remove('hide');
      }
    } catch (_) {}
    $('#btn-copy-sub').onclick = () => { navigator.clipboard.writeText(text); toast('copied'); };
  }
  async function refreshPending() {
    try {
      const data = await (await api('/api/edge/pending')).json();
      const list = data.pending || [];
      const box = $('#edge-pending');
      if (!box) return;
      if (!list.length) { box.innerHTML = '<div class="muted">No pending devices</div>'; return; }
      box.innerHTML = '<h3>Pending approval</h3>' + list.map((d) => {
        const id = d.device_id || '';
        return `<div class="row"><code>${id}</code> ${d.status||''} ${d.board||''} ${d.wan_ip||''} ${d.hostname||''}
          <button class="primary btn-appr" data-id="${id}">Approve</button>
          <button class="ghost btn-deny" data-id="${id}">Deny</button></div>`;
      }).join('');
      box.querySelectorAll('.btn-appr').forEach((b) => b.onclick = async () => {
        await api('/api/edge/approve', { method: 'POST', body: JSON.stringify({ device_id: b.dataset.id }) });
        toast('approved'); refreshPending(); refreshRouters();
      });
      box.querySelectorAll('.btn-deny').forEach((b) => b.onclick = async () => {
        await api('/api/edge/deny', { method: 'POST', body: JSON.stringify({ device_id: b.dataset.id }) });
        toast('denied'); refreshPending();
      });
    } catch (e) { console.warn(e); }
  }
  async function refreshRouters() {
    refreshPending();
    const data = await (await api('/api/edge/devices')).json();
    const list = data.devices || data || [];
    const tb = $('#routers-table tbody'); tb.innerHTML = '';
    list.forEach((d) => {
      const tr = document.createElement('tr');
      const id = d.device_id || d.id || '';
      tr.innerHTML = `<td>${id}</td><td>${d.status||''}</td><td>${d.hostname||''}</td><td>${d.wan_ip||''}</td>
        <td>
          <button class="ghost btn-ping" data-id="${id}">ping</button>
          <button class="ghost btn-apply" data-id="${id}">apply</button>
          <button class="ghost btn-gstat" data-id="${id}">guest status</button>
          <button class="ghost btn-ggrant" data-id="${id}">guest grant</button>
          <button class="ghost btn-rev" data-id="${id}">revoke</button>
        </td>`;
      tb.appendChild(tr);
    });
    tb.querySelectorAll('.btn-ping').forEach((b) => b.onclick = async () => {
      await api('/api/edge/cmd', { method: 'POST', body: JSON.stringify({ device_id: b.dataset.id, action: 'ping' }) });
      toast('queued');
    });
    tb.querySelectorAll('.btn-apply').forEach((b) => b.onclick = async () => {
      await api('/api/edge/cmd', { method: 'POST', body: JSON.stringify({ device_id: b.dataset.id, action: 'apply_template' }) });
      toast('apply queued');
    });
    tb.querySelectorAll('.btn-rev').forEach((b) => b.onclick = async () => {
      await api('/api/edge/revoke', { method: 'POST', body: JSON.stringify({ device_id: b.dataset.id }) });
      toast('revoked'); refreshRouters();
    });
    tb.querySelectorAll('.btn-gstat').forEach((b) => b.onclick = async () => {
      await api('/api/edge/guest/status?device_id=' + encodeURIComponent(b.dataset.id));
      toast('guest status queued');
    });
    tb.querySelectorAll('.btn-ggrant').forEach((b) => b.onclick = async () => {
      const code = prompt('Guest code (or MAC)');
      if (!code) return;
      const minutes = parseInt(prompt('Minutes (1-1440)', '10') || '10', 10);
      await api('/api/edge/guest/grant', { method: 'POST', body: JSON.stringify({ device_id: b.dataset.id, code, minutes }) });
      toast('guest grant queued');
    });
    try {
      const res = await (await api('/api/edge/results')).json();
      $('#edge-results').textContent = JSON.stringify(res.results || res, null, 2);
      if (list.length) {
        const id0 = list[0].device_id || list[0].id;
        const bk = await (await api('/api/edge/backups?device_id=' + encodeURIComponent(id0))).json();
        $('#edge-backups').textContent = 'backups ' + id0 + ':\n' + JSON.stringify(bk.backups || bk, null, 2);
      }
    } catch (_) {}
  }
  
  function formToTemplate() {
    return {
      id: $('#f-id').value.trim() || 'default',
      role: $('#f-role').value.trim() || 'site',
      network: {
        lan_ip: $('#f-lan-ip').value.trim(),
        lan_mask: $('#f-lan-mask').value.trim() || '255.255.255.0',
        dhcp: $('#f-dhcp').checked,
      },
      wifi: {
        ssid: $('#f-ssid').value.trim(),
        key: $('#f-wkey').value,
        encryption: $('#f-enc').value,
      },
      vpn: {
        enabled: $('#f-vpn').checked,
        mode: $('#f-vpn-mode').value,
      },
      guest: {
        enabled: $('#f-guest') ? $('#f-guest').checked : false,
        ssid: $('#f-guest-ssid') ? $('#f-guest-ssid').value.trim() : 'Guest',
        desk_pin: $('#f-guest-pin') ? $('#f-guest-pin').value : '',
        hidden: true,
      },
    };
  }
  function templateToForm(t) {
    if (!t) return;
    $('#f-id').value = t.id || 'default';
    $('#f-role').value = t.role || 'site';
    const net = t.network || {};
    $('#f-lan-ip').value = net.lan_ip || '';
    $('#f-lan-mask').value = net.lan_mask || '255.255.255.0';
    $('#f-dhcp').checked = net.dhcp !== false;
    const wifi = t.wifi || {};
    $('#f-ssid').value = wifi.ssid || '';
    $('#f-wkey').value = wifi.key || '';
    if (wifi.encryption) $('#f-enc').value = wifi.encryption;
    const vpn = t.vpn || {};
    $('#f-vpn').checked = vpn.enabled !== false && vpn.enabled !== 'false';
    if (vpn.mode) $('#f-vpn-mode').value = vpn.mode;
    $('#tmpl-editor').value = JSON.stringify(t, null, 2);
  }

  async function refreshTemplates() {
    const data = await (await api('/api/edge/templates')).json();
    const list = data.templates || data || [];
    $('#templates-raw').textContent = JSON.stringify(list, null, 2);
    if (list.length) {
      const id = ($('#f-id') && $('#f-id').value) || 'default';
      const found = list.find((x) => x.id === id) || list[0];
      if (found && !$('#f-lan-ip').value) templateToForm(found);
      if ($('#tmpl-editor') && !$('#tmpl-editor').value) $('#tmpl-editor').value = JSON.stringify(found, null, 2);
    }
  }
  async function refreshMetrics() {
    const m = await (await api('/api/metrics')).json();
    $('#metrics-raw').textContent = JSON.stringify(m, null, 2);
  }
  async function refreshProbes() {
    const p = await (await api('/api/probes')).json();
    $('#probes-raw').textContent = JSON.stringify(p, null, 2);
  }
  document.getElementById('btn-git-refresh')?.addEventListener('click', () => refreshGit());
document.getElementById('btn-git-delete')?.addEventListener('click', async () => {
  const name = window._gitSelected;
  if (!name || !confirm('Delete ' + name + '?')) return;
  await api('/api/git/repos?name=' + encodeURIComponent(name), { method: 'DELETE' });
  window._gitSelected = '';
  refreshGit();
});
document.getElementById('btn-git-init')?.addEventListener('click', async () => {
  const name = document.getElementById('git-new-name')?.value?.trim();
  if (!name) return;
  await api('/api/git/repos', { method: 'POST', body: JSON.stringify({ name }) });
  refreshGit();
});
document.getElementById('btn-git-show')?.addEventListener('click', async () => {
  const name = window._gitSelected;
  if (!name) return;
  const rev = document.getElementById('git-rev')?.value || 'HEAD';
  const data = await (await api('/api/git/show?name=' + encodeURIComponent(name) + '&rev=' + encodeURIComponent(rev))).json();
  const el = document.getElementById('git-show');
  if (el) el.textContent = data.show || data.error || '';
});
document.getElementById('btn-git-run')?.addEventListener('click', async () => {
  const name = window._gitSelected;
  const pipe = document.getElementById('git-pipeline-sel')?.value;
  if (!name || !pipe) return;
  const data = await (await api('/api/git/pipeline', { method: 'POST', body: JSON.stringify({ repo: name, pipeline: pipe }) })).json();
  const el = document.getElementById('git-pipeline-out');
  if (el) el.textContent = data.output || data.error || JSON.stringify(data);
});
function refreshAll() {
    refreshNodes().catch(()=>{});
    refreshOverview(); refreshUsers(); refreshRouters(); refreshTemplates(); refreshMetrics(); refreshProbes(); refreshAddons();
  }

  $('#btn-login').onclick = async () => {
    state.token = $('#token').value.trim();
    try {
      const r = await api('/api/session');
      if (!r.ok) throw new Error('bad token');
      localStorage.setItem('nd_token', state.token);
      showDash();
    } catch (e) { $('#login-err').textContent = 'Invalid session'; }
  };
  if ($('#btn-lp-refresh')) $('#btn-lp-refresh').onclick = () => refreshAddons();
  if ($('#btn-relay-export')) $('#btn-relay-export').onclick = () => refreshRelay();
  if ($('#btn-relay-exit-on')) $('#btn-relay-exit-on').onclick = async () => { await api('/api/relay/exit?enabled=1', {method:'POST'}); refreshRelay(); };
  if ($('#btn-relay-exit-off')) $('#btn-relay-exit-off').onclick = async () => { await api('/api/relay/exit?enabled=0', {method:'POST'}); refreshRelay(); };
  $('#btn-logout').onclick = logout;
  $('#btn-lang').onclick = () => {
    state.lang = state.lang === 'en' ? 'ru' : 'en';
    localStorage.setItem('nd_lang', state.lang); applyI18n();
  };
  document.querySelectorAll('.tab').forEach((b) => b.onclick = () => tab(b.dataset.tab));
  $('#btn-refresh-users').onclick = refreshUsers;
  $('#btn-add-user').onclick = async () => {
    const name = $('#new-user').value.trim();
    const note = $('#new-note').value.trim();
    if (!name) return;
    await api('/vpn/users', { method: 'POST', body: JSON.stringify({ name, note }) });
    $('#new-user').value = ''; refreshUsers(); toast('added');
  };
  $('#btn-refresh-tmpl').onclick = refreshTemplates;
  $('#btn-form-save').onclick = async () => {
    const body = formToTemplate();
    await api('/api/edge/templates', { method: 'POST', body: JSON.stringify(body) });
    toast('saved ' + body.id); refreshTemplates();
  };
  $('#btn-form-load').onclick = async () => {
    const data = await (await api('/api/edge/templates')).json();
    const list = data.templates || [];
    const id = $('#f-id').value.trim() || 'default';
    const found = list.find((x) => x.id === id) || list[0];
    templateToForm(found);
    toast(found ? 'loaded' : 'empty');
  };
  $('#btn-load-tmpl').onclick = async () => {
    const id = $('#f-id').value.trim() || 'default';
    const data = await (await api('/api/edge/templates')).json();
    const list = data.templates || [];
    const found = list.find((t) => t.id === id) || list[0];
    if (found) {
      $('#f-id').value = found.id || id;
      $('#tmpl-editor').value = JSON.stringify(found, null, 2);
    } else toast('not found');
  };
  $('#btn-save-tmpl').onclick = async () => {
    try {
      const body = JSON.parse($('#tmpl-editor').value);
      body.id = $('#f-id').value.trim() || body.id || 'default';
      await api('/api/edge/templates', { method: 'POST', body: JSON.stringify(body) });
      toast('saved'); refreshTemplates();
    } catch (e) { toast('JSON error: ' + e.message); }
  };
  $('#btn-save-default').onclick = async () => {
    await api('/api/edge/templates', { method: 'POST', body: JSON.stringify({
      id: 'default', role: 'site',
      network: { lan_ip: '192.168.50.1', lan_mask: '255.255.255.0', dhcp: true },
      wifi: { ssid: 'Netductor', encryption: 'psk2', key: '' },
      vpn: { enabled: true }
    })});
    toast('default saved'); refreshTemplates();
  };
  $('#btn-bind').onclick = async () => {
    const device_id = $('#bind-device').value.trim();
    const template_id = $('#bind-tmpl').value.trim() || 'default';
    const overlay = {};
    if ($('#bind-ssid').value.trim()) overlay.ssid = $('#bind-ssid').value.trim();
    if ($('#bind-lan').value.trim()) overlay.lan_ip = $('#bind-lan').value.trim();
    await api('/api/edge/bind-template', { method: 'POST', body: JSON.stringify({ device_id, template_id, overlay }) });
    toast('bound');
  };
  applyI18n();
  if (state.token) showDash();
})();

  document.getElementById('btn-refresh-nodes')?.addEventListener('click', () => refreshNodes().catch(e=>toast(e.message)));
  document.getElementById('btn-node-rename')?.addEventListener('click', async () => {
    const id = document.getElementById('node-id').value.trim();
    const hostname = document.getElementById('node-hn').value.trim();
    await api('/api/nodes/hostname', { method: 'POST', body: JSON.stringify({ id, hostname }) });
    toast('desired hostname set');
    refreshNodes();
  });

document.getElementById('btn-node-upgrade')?.addEventListener('click', async () => {
  const id = document.getElementById('node-id')?.value?.trim();
  if (!id) return toast('select node');
  await api('/api/relay/cmd', { method: 'POST', body: JSON.stringify({ id, cmd: 'upgrade' }) });
  toast('upgrade queued');
});
document.getElementById('btn-node-reboot')?.addEventListener('click', async () => {
  const id = document.getElementById('node-id')?.value?.trim();
  if (!id) return toast('select node');
  await api('/api/relay/cmd', { method: 'POST', body: JSON.stringify({ id, cmd: 'reboot' }) });
  toast('reboot queued');
});
document.getElementById('btn-relay-sync')?.addEventListener('click', async () => {
  await api('/api/relay/sync', { method: 'POST' });
  toast('sync bumped');
  refreshRelay?.();
});

async function refreshAlerts() {
  try {
    const r = await api('/api/dashboard');
    const d = await r.json();
    const el = document.getElementById('alerts-box');
    if (!el) return;
    const probes = d.probes || [];
    const bad = probes.filter(p => p.ok === false);
    if (bad.length === 0) {
      el.innerHTML = '<span class="ok">No probe failures</span>';
      return;
    }
    el.innerHTML = bad.map(p => `<div class="alert">⚠️ ${p.name}: ${p.error||'fail'}</div>`).join('');
  } catch (e) { /* ignore */ }
}
document.getElementById('btn-refresh-alerts')?.addEventListener('click', () => refreshAlerts());
setInterval(() => { try { refreshAlerts(); } catch(_){} }, 60000);

document.getElementById('btn-journal')?.addEventListener('click', async () => {
  const unit = document.getElementById('ops-unit')?.value || 'sing-box';
  try {
    const r = await api('/api/nodes/journal?unit=' + encodeURIComponent(unit));
    const d = await r.json();
    document.getElementById('ops-out').textContent = d.log || JSON.stringify(d, null, 2);
  } catch (e) { toast(e.message); }
});
document.getElementById('btn-restart-svc')?.addEventListener('click', async () => {
  const unit = document.getElementById('ops-unit')?.value || 'sing-box';
  try {
    const r = await api('/api/nodes/restart-service', { method: 'POST', body: JSON.stringify({ unit }) });
    const d = await r.json();
    document.getElementById('ops-out').textContent = JSON.stringify(d, null, 2);
    toast(d.ok ? 'restarted' : 'failed');
  } catch (e) { toast(e.message); }
});
document.getElementById('btn-relay-sync')?.addEventListener('click', async () => {
  try {
    await api('/api/relay/export', { method: 'POST' });
    toast('relay export/sync requested');
    refreshRelay?.();
  } catch (e) { toast(e.message); }
});

document.getElementById('btn-node-journal')?.addEventListener('click', async () => {
  const id = document.getElementById('node-id')?.value?.trim();
  const unit = 'sing-box';
  const q = id ? ('?id=' + encodeURIComponent(id) + '&unit=' + unit) : ('?unit=' + unit);
  try {
    const r = await api('/api/nodes/journal' + q);
    const d = await r.json();
    const box = document.getElementById('ops-out') || document.getElementById('sni-box');
    if (box) box.textContent = d.log || JSON.stringify(d, null, 2);
    toast(d.queued ? 'journal queued' : 'journal');
  } catch (e) { toast(e.message); }
});
document.getElementById('btn-sni-refresh')?.addEventListener('click', async () => {
  try {
    const r = await api('/api/sni');
    const d = await r.json();
    document.getElementById('sni-box').textContent = 'active: ' + (d.active||'') + '\n\n' + JSON.stringify(d.presets || d, null, 2);
  } catch (e) { toast(e.message); }
});
document.getElementById('btn-sni-apply')?.addEventListener('click', async () => {
  const name = document.getElementById('sni-pick')?.value?.trim();
  if (!name) { toast('pick preset'); return; }
  try {
    const r = await api('/api/sni', { method: 'POST', body: JSON.stringify({ name }) });
    const d = await r.json();
    toast('SNI → ' + (d.active || name));
    document.getElementById('btn-sni-refresh')?.click();
  } catch (e) { toast(e.message); }
});


async function refreshStatusPanel() {
  try {
    const [st, nodes, users] = await Promise.all([
      api('/api/status').then(r => r.json()).catch(() => ({})),
      api('/api/nodes').then(r => r.json()).catch(() => ({})),
      api('/api/vpn/users').then(r => r.json()).catch(() => ({})),
    ]);
    const lines = [];
    lines.push('=== core ===');
    lines.push(JSON.stringify(st, null, 2));
    lines.push('=== nodes ===');
    const nl = nodes.nodes || nodes || [];
    (Array.isArray(nl) ? nl : []).forEach(n => {
      lines.push(`${n.status||'?'} ${n.hostname||n.id} role=${n.role} ip=${n.public_ip||n.ip||''}`);
    });
    lines.push('=== vpn users ===');
    const ul = users.users || users || [];
    (Array.isArray(ul) ? ul : []).forEach(u => {
      lines.push(`${u.name||u} ${u.enabled===false?'off':'on'}`);
    });
    const box = document.getElementById('status-box');
    if (box) box.textContent = lines.join('\n');
  } catch (e) { toast(e.message); }
}
document.getElementById('btn-refresh-status')?.addEventListener('click', () => refreshStatusPanel());

async function refreshSites() {
  try {
    const r = await api('/api/sites');
    const d = await r.json();
    document.getElementById('sites-list').textContent = JSON.stringify(d.sites || d, null, 2);
  } catch (e) { toast(e.message); }
}
document.getElementById('btn-sites-refresh')?.addEventListener('click', () => refreshSites());
document.getElementById('btn-site-save')?.addEventListener('click', async () => {
  const body = {
    id: document.getElementById('site-id')?.value?.trim(),
    name: document.getElementById('site-name')?.value?.trim(),
    rpi_id: document.getElementById('site-rpi')?.value?.trim(),
    mikrotik_id: document.getElementById('site-mt')?.value?.trim(),
  };
  try {
    await api('/api/sites', { method: 'POST', body: JSON.stringify(body) });
    toast('site saved');
    refreshSites();
  } catch (e) { toast(e.message); }
});
document.getElementById('btn-site-rsc')?.addEventListener('click', async () => {
  const id = document.getElementById('site-id')?.value?.trim();
  try {
    const r = await api('/api/sites/rsc?id=' + encodeURIComponent(id || ''));
    const d = await r.json();
    document.getElementById('site-out').textContent = d.rsc || JSON.stringify(d, null, 2);
  } catch (e) { toast(e.message); }
});


async function refreshSSHHosts() {
  try {
    const r = await api('/api/ssh-hosts');
    const d = await r.json();
    const el = document.getElementById('sshhosts-box');
    if (el) el.textContent = JSON.stringify(d, null, 2);
  } catch (e) {
    const el = document.getElementById('sshhosts-box');
    if (el) el.textContent = String(e);
  }
}
document.getElementById('btn-sshhosts-refresh')?.addEventListener('click', () => refreshSSHHosts());
document.getElementById('btn-sshhosts-forget')?.addEventListener('click', async () => {
  const id = document.getElementById('sshhosts-id')?.value?.trim();
  if (!id) return;
  const kind = document.getElementById('sshhosts-kind')?.value || '';
  let q = '/api/ssh-hosts?id=' + encodeURIComponent(id);
  if (kind) q += '&kind=' + encodeURIComponent(kind);
  await api(q, { method: 'DELETE' });
  refreshSSHHosts();
});
document.getElementById('btn-sshhosts-clear-mt')?.addEventListener('click', async () => {
  if (!confirm('Clear all MikroTik TOFU keys?')) return;
  await api('/api/ssh-hosts/clear?kind=mt', { method: 'POST' });
  refreshSSHHosts();
});
document.getElementById('btn-sshhosts-clear-relay')?.addEventListener('click', async () => {
  if (!confirm('Clear all relay TOFU keys?')) return;
  await api('/api/ssh-hosts/clear?kind=relay', { method: 'POST' });
  refreshSSHHosts();
});

document.querySelectorAll('.tab').forEach((btn) => {
  btn.addEventListener('click', () => {
    if (btn.getAttribute('data-tab') === 'sshhosts') refreshSSHHosts();
  });
});

document.getElementById('btn-vpn-rename')?.addEventListener('click', async () => {
  const name = document.getElementById('detail-title')?.textContent?.trim();
  const neu = document.getElementById('vpn-rename-to')?.value?.trim();
  if (!name || !neu) return toast('name required');
  const r = await api('/vpn/users/' + encodeURIComponent(name) + '/rename', {
    method: 'POST', body: JSON.stringify({ new_name: neu })
  });
  if (!r.ok) { toast('rename failed'); return; }
  toast('renamed → ' + neu);
  document.getElementById('vpn-rename-to').value = '';
  if (typeof refreshUsers === 'function') refreshUsers();
});
async function refreshBackup() {
  const el = document.getElementById('backup-status');
  if (!el) return;
  try {
    const r = await (await api('/api/backup/peer')).json();
    el.textContent = r.status || JSON.stringify(r);
  } catch (e) { el.textContent = String(e); }
}
document.getElementById('btn-backup-peer')?.addEventListener('click', async () => {
  const target = document.getElementById('backup-target')?.value?.trim();
  if (!target) return toast('target required');
  const r = await api('/api/backup/peer', { method: 'POST', body: JSON.stringify({ target }) });
  if (!r.ok) toast('failed'); else toast('peer set');
  refreshBackup();
});
document.getElementById('btn-backup-run')?.addEventListener('click', async () => {
  const r = await api('/api/backup/run', { method: 'POST', body: '{}' });
  const j = await r.json().catch(() => ({}));
  toast(j.path || j.error || (r.ok ? 'ok' : 'fail'));
  refreshBackup();
});
document.querySelectorAll('.tab').forEach((btn) => {
  btn.addEventListener('click', () => {
    if (btn.getAttribute('data-tab') === 'backup') refreshBackup();
  });
});

async function refreshAudit() {
  const el = document.getElementById('audit-box');
  if (!el) return;
  try {
    const j = await (await api('/api/audit')).json();
    const lines = (j.events || []).map((e) => `${e.ts}\t${e.actor||''}\t${e.action}\t${e.target||''}\t${e.detail||''}`);
    el.textContent = lines.join('\n') || '(empty)';
  } catch (e) { el.textContent = String(e); }
}
async function refreshSessions() {
  const el = document.getElementById('sessions-box');
  if (!el) return;
  try {
    const j = await (await api('/api/sessions')).json();
    const lines = (j.sessions || []).map((s) => `${s.id}\texp=${s.exp}\t${s.label||''}\t${s.ip||''}`);
    el.textContent = lines.join('\n') || '(none)';
  } catch (e) { el.textContent = String(e); }
}
document.getElementById('btn-audit-refresh')?.addEventListener('click', refreshAudit);
document.getElementById('btn-sessions-refresh')?.addEventListener('click', refreshSessions);
document.getElementById('btn-sessions-revoke')?.addEventListener('click', async () => {
  await api('/api/sessions?action=revoke-all', { method: 'POST', body: '{}' });
  toast('revoked');
  refreshSessions();
});
document.getElementById('btn-vpn-refresh-links')?.addEventListener('click', async () => {
  const r = await api('/api/vpn/refresh-links', { method: 'POST', body: '{}' });
  const j = await r.json().catch(() => ({}));
  toast('refreshed ' + (j.refreshed ?? '?'));
});
document.querySelectorAll('.tab').forEach((btn) => {
  btn.addEventListener('click', () => {
    const t = btn.getAttribute('data-tab');
    if (t === 'audit') refreshAudit();
    if (t === 'sessions') refreshSessions();
  });
});


  function wireEdgeRecovery() {
    const btnR = document.getElementById('btn-edge-recovery');
    if (btnR) btnR.onclick = async () => {
      const site = document.getElementById('rec-site')?.value || '';
      const note = document.getElementById('rec-note')?.value || '';
      const data = await (await api('/api/edge/recovery', { method: 'POST', body: JSON.stringify({ site_id: site, note }) })).json();
      const out = document.getElementById('edge-recovery-out');
      if (out) out.textContent = JSON.stringify(data, null, 2);
      if (data.code) {
        try { await navigator.clipboard.writeText(data.code); toast('code copied'); } catch (_) { toast('code issued'); }
      }
    };
    const btnReg = document.getElementById('btn-edge-register');
    if (btnReg) btnReg.onclick = async () => {
      const device_id = document.getElementById('reg-device')?.value || '';
      const site_id = document.getElementById('reg-site')?.value || '';
      await api('/api/edge/register', { method: 'POST', body: JSON.stringify({ device_id, site_id }) });
      toast('registered pending'); refreshPending(); refreshRouters();
    };
    const btnSS = document.getElementById('btn-edge-setsite');
    if (btnSS) btnSS.onclick = async () => {
      const device_id = document.getElementById('setsite-device')?.value || '';
      const site_id = document.getElementById('setsite-site')?.value || '';
      await api('/api/edge/set-site', { method: 'POST', body: JSON.stringify({ device_id, site_id }) });
      toast('site set');
    };
    const btnEx = document.getElementById('btn-edge-export');
    if (btnEx) btnEx.onclick = async () => {
      const data = await (await api('/api/edge/export')).json();
      const out = document.getElementById('edge-export-out');
      if (out) out.textContent = JSON.stringify(data, null, 2);
      toast('exported');
    };
  }

  async function refreshMtls() {
    const pre = document.getElementById('mtls-certs');
    if (!pre) return;
    try {
      const data = await (await api('/api/mtls/certs')).json();
      pre.textContent = JSON.stringify(data, null, 2);
    } catch (e) { pre.textContent = String(e); }
  }
  document.getElementById('btn-mtls-refresh')?.addEventListener('click', refreshMtls);


async function refreshGit() {
  try {
    const data = await (await api('/api/git/repos')).json();
    const ul = document.getElementById('git-repo-list');
    if (!ul) return;
    ul.innerHTML = '';
    window._gitSelected = window._gitSelected || '';
    (data.repos || []).forEach((r) => {
      const li = document.createElement('li');
      const name = r.name || r;
      li.innerHTML = `<button type="button" class="ghost git-pick" data-name="${name}">${name}</button>`;
      ul.appendChild(li);
    });
    ul.querySelectorAll('.git-pick').forEach((b) => b.onclick = async () => {
      window._gitSelected = b.dataset.name;
      const log = await (await api('/api/git/log?name=' + encodeURIComponent(b.dataset.name) + '&n=30')).json();
      const el = document.getElementById('git-log');
      if (el) el.textContent = log.log || log.error || '';
    });
    const pl = await (await api('/api/git/pipelines')).json();
    const sel = document.getElementById('git-pipeline-sel');
    if (sel) {
      sel.innerHTML = '';
      (pl.pipelines || []).forEach((n) => {
        const o = document.createElement('option');
        o.value = n; o.textContent = n;
        sel.appendChild(o);
      });
    }
  } catch (e) { console.warn(e); }
}
