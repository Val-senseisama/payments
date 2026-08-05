// PayEngine Client Logic - Monochrome Theme
document.addEventListener('DOMContentLoaded', () => {
  
  // State
  let state = {
    useLiveAPI: false,
    apiBaseURL: 'http://localhost:8080/api/v1',
    transactions: [
      { id: 'tx_01h92k1', reference: 'TXN-PAY-9842', type: 'payment_in', amount: 500000, status: 'completed', created_at: '2026-08-05 10:14:22' },
      { id: 'tx_01h92k2', reference: 'TXN-TRANSFER-104', type: 'transfer', amount: 1250000, status: 'completed', created_at: '2026-08-05 09:42:10' },
      { id: 'tx_01h92k3', reference: 'TXN-PAY-FAIL-88', type: 'payment_out', amount: 75000, status: 'failed', created_at: '2026-08-05 08:11:05' },
    ],
    accounts: [
      { id: 'acc_01', name: 'Operating Cash Account', type: 'asset', balance: 142500000, created_at: '2026-08-01' },
      { id: 'acc_02', name: 'Customer Clearing Account', type: 'liability', balance: 0, created_at: '2026-08-01' },
      { id: 'acc_03', name: 'Payment Processing Fee Revenue', type: 'revenue', balance: 8500000, created_at: '2026-08-01' },
      { id: 'acc_04', name: 'Gateway Processing Expense', type: 'expense', balance: 1200000, created_at: '2026-08-01' },
    ],
    ledger: [
      { id: 'ent_01', txn_ref: 'TXN-PAY-9842', account: 'Operating Cash Account', direction: 'debit', amount: 500000, timestamp: '2026-08-05 10:14:22' },
      { id: 'ent_02', txn_ref: 'TXN-PAY-9842', account: 'Payment Processing Fee Revenue', direction: 'credit', amount: 500000, timestamp: '2026-08-05 10:14:22' },
    ],
    auditLogs: [
      { id: 'log_991', entity_type: 'payment', action: 'payment.processed', user: 'admin@acme.com', timestamp: '2026-08-05 10:14:22' },
      { id: 'log_990', entity_type: 'company', action: 'company.seeded_accounts', user: 'system', timestamp: '2026-08-01 00:00:00' },
    ]
  };

  // DOM Elements
  const navItems = document.querySelectorAll('.nav-item');
  const tabContents = document.querySelectorAll('.tab-content');
  const pageTitle = document.getElementById('page-title');
  const pageSubtitle = document.getElementById('page-subtitle');
  const paymentForm = document.getElementById('payment-form');
  const jsonBlock = document.getElementById('response-json-block');
  const responseStatus = document.getElementById('response-status');
  const apiToggleBtn = document.getElementById('api-toggle-btn');

  // Tab Titles Map
  const tabInfo = {
    payments: { title: 'Payment Terminal', subtitle: 'Execute payment processing charges through PSP adapters with Redis locking and SSE updates.' },
    transactions: { title: 'Transactions & SSE Stream', subtitle: 'Live transaction log streaming events directly via Server-Sent Events.' },
    ledger: { title: 'General Ledger', subtitle: 'Multi-tenant double-entry accounting records with zero variance debit/credit checks.' },
    accounts: { title: 'Chart of Accounts', subtitle: 'Configure asset, liability, revenue, and expense accounts for your company.' },
    audit: { title: 'System Audit Logs', subtitle: 'Immutable, non-repudiable audit trails of all mutating platform actions.' }
  };

  // Navigation Logic
  navItems.forEach(item => {
    item.addEventListener('click', () => {
      const targetTab = item.getAttribute('data-tab');
      
      navItems.forEach(n => n.classList.remove('active'));
      tabContents.forEach(c => c.style.display = 'none');

      item.classList.add('active');
      document.getElementById(`tab-${targetTab}`).style.display = 'block';

      if (tabInfo[targetTab]) {
        pageTitle.textContent = tabInfo[targetTab].title;
        pageSubtitle.textContent = tabInfo[targetTab].subtitle;
      }

      renderTables();
    });
  });

  // API Toggle Button
  apiToggleBtn.addEventListener('click', () => {
    state.useLiveAPI = !state.useLiveAPI;
    if (state.useLiveAPI) {
      apiToggleBtn.textContent = 'Live API';
    } else {
      apiToggleBtn.textContent = 'Mock Mode';
    }
  });

  // Handle Payment Execution
  paymentForm.addEventListener('submit', async (e) => {
    e.preventDefault();

    const ref = document.getElementById('pay-reference').value;
    const psp = document.getElementById('pay-psp').value;
    const amount = parseInt(document.getElementById('pay-amount').value, 10);
    const failSim = document.getElementById('pay-fail-sim').value;

    let finalRef = ref;
    if (failSim === 'fail' && !ref.toLowerCase().includes('fail')) {
      finalRef = `${ref}-FAIL`;
    }

    responseStatus.className = 'badge badge-warning';
    responseStatus.textContent = 'Processing';
    jsonBlock.textContent = `// Acquiring Redis lock: 'lock:payment:${finalRef}'...\n// Dispatching charge to ${psp.toUpperCase()} adapter...`;

    setTimeout(() => {
      if (failSim === 'fail') {
        // Failure Simulation
        responseStatus.className = 'badge badge-danger';
        responseStatus.textContent = 'Failed (402)';
        
        const failResponse = {
          status: 'failed',
          psp: psp,
          error_code: 'insufficient_funds',
          message: 'Payment declined by issuer',
          transaction_reference: finalRef,
          amount_kobo: amount,
          attempt_id: `att_${Math.random().toString(36).substr(2, 9)}`,
          redis_lock: 'released',
          timestamp: new Date().toISOString()
        };

        jsonBlock.textContent = JSON.stringify(failResponse, null, 2);

        state.transactions.unshift({
          id: `tx_${Math.random().toString(36).substr(2, 8)}`,
          reference: finalRef,
          type: 'payment_in',
          amount: amount,
          status: 'failed',
          created_at: new Date().toLocaleString()
        });

      } else {
        // Success Path
        responseStatus.className = 'badge badge-success';
        responseStatus.textContent = 'Success (200)';

        const successResponse = {
          status: 'success',
          psp: psp,
          external_reference: `MOCK-REF-${Math.floor(100000 + Math.random() * 900000)}`,
          transaction_reference: finalRef,
          amount_kobo: amount,
          attempt_id: `att_${Math.random().toString(36).substr(2, 9)}`,
          ledger_job_enqueued: true,
          sse_broadcast_event: 'payment.succeeded',
          timestamp: new Date().toISOString()
        };

        jsonBlock.textContent = JSON.stringify(successResponse, null, 2);

        const newTxnId = `tx_${Math.random().toString(36).substr(2, 8)}`;
        state.transactions.unshift({
          id: newTxnId,
          reference: finalRef,
          type: 'payment_in',
          amount: amount,
          status: 'completed',
          created_at: new Date().toLocaleString()
        });

        // Auto post double-entry ledger
        state.ledger.unshift({
          id: `ent_${Math.random().toString(36).substr(2, 6)}`,
          txn_ref: finalRef,
          account: 'Operating Cash Account',
          direction: 'debit',
          amount: amount,
          timestamp: new Date().toLocaleString()
        });

        state.ledger.unshift({
          id: `ent_${Math.random().toString(36).substr(2, 6)}`,
          txn_ref: finalRef,
          account: 'Customer Clearing Account',
          direction: 'credit',
          amount: amount,
          timestamp: new Date().toLocaleString()
        });

        state.auditLogs.unshift({
          id: `log_${Math.floor(1000 + Math.random() * 9000)}`,
          entity_type: 'payment',
          action: 'payment.processed',
          user: 'admin@acme.com',
          timestamp: new Date().toLocaleString()
        });
      }

      renderTables();
    }, 450);
  });

  // Render Tables Logic
  function renderTables() {
    // 1. Transactions Table
    const txBody = document.getElementById('transactions-tbody');
    txBody.innerHTML = state.transactions.map(t => `
      <tr>
        <td><strong>${t.reference}</strong><br><span style="font-size:10px;color:var(--text-subtle);">${t.id}</span></td>
        <td><span class="badge badge-info">${t.type}</span></td>
        <td><strong>₦${(t.amount / 100).toLocaleString(undefined, {minimumFractionDigits: 2})}</strong></td>
        <td><span class="badge ${t.status === 'completed' ? 'badge-success' : 'badge-danger'}">${t.status}</span></td>
        <td>${t.created_at}</td>
      </tr>
    `).join('');

    // 2. Ledger Table
    const ledgerBody = document.getElementById('ledger-tbody');
    ledgerBody.innerHTML = state.ledger.map(l => `
      <tr>
        <td><code>${l.id}</code></td>
        <td><strong>${l.txn_ref}</strong></td>
        <td>${l.account}</td>
        <td><span class="badge ${l.direction === 'debit' ? 'badge-success' : 'badge-info'}">${l.direction}</span></td>
        <td><strong>₦${(l.amount / 100).toLocaleString(undefined, {minimumFractionDigits: 2})}</strong></td>
        <td>${l.timestamp}</td>
      </tr>
    `).join('');

    // 3. Accounts Table
    const accBody = document.getElementById('accounts-tbody');
    accBody.innerHTML = state.accounts.map(a => `
      <tr>
        <td><code>${a.id}</code></td>
        <td><strong>${a.name}</strong></td>
        <td><span class="badge badge-info">${a.type}</span></td>
        <td><strong>₦${(a.balance / 100).toLocaleString(undefined, {minimumFractionDigits: 2})}</strong></td>
        <td>${a.created_at}</td>
      </tr>
    `).join('');

    // 4. Audit Table
    const auditBody = document.getElementById('audit-tbody');
    auditBody.innerHTML = state.auditLogs.map(au => `
      <tr>
        <td><code>${au.id}</code></td>
        <td><span class="badge badge-warning">${au.entity_type}</span></td>
        <td><strong>${au.action}</strong></td>
        <td>${au.user}</td>
        <td>${au.timestamp}</td>
      </tr>
    `).join('');
  }

  // Initial render
  renderTables();
});
