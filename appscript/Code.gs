var PROP_SECRET = 'SHARED_SECRET';
var PROP_WEBHOOK = 'BACKEND_WEBHOOK_URL';
var PROP_FORMS = 'gform:forms';
var FORM_PREFIX = 'gform:form:';
var SECRET_HEADER = 'X-Gform-Secret';
var SYNC_INTERVAL_MINUTES = 1;

function doPost(e) {
  try {
    var body = JSON.parse(e.postData.contents);

    requireSecret_(body.secret);

    switch (body.action) {
      case 'createForm':
        return json_({ ok: true, data: createForm(body.payload) });
      case 'closeForm':
        return json_({ ok: true, data: closeForm(body.payload.form_id) });
      case 'ping':
        return json_({ ok: true, data: { ok: true } });
      default:
        return json_({ ok: false, error: 'неизвестное действие: ' + body.action });
    }
  } catch (err) {
    return json_({ ok: false, error: String(err && err.message ? err.message : err) });
  }
}

function doGet(e) {
  var formId = e && e.parameter ? e.parameter.form : '';
  var meta = formId ? readForm_(formId) : null;

  if (!meta) {
    return errorPage_('Ссылка недействительна', 'Проверьте ссылку или обратитесь к администратору.');
  }

  var form;
  try {
    form = FormApp.openById(formId);
  } catch (err) {
    return errorPage_('Ссылка недействительна', 'Форма не найдена в Google Drive.');
  }

  if (!form.isAcceptingResponses()) {
    return errorPage_('Тест закрыт', 'Эта форма больше не принимает ответы.');
  }

  var template = HtmlService.createTemplateFromFile('Index');
  template.config = JSON.stringify({
    formId: formId,
    title: meta.title,
    description: meta.description,
    durationMinutes: meta.duration_minutes,
    branches: meta.branches,
    questions: meta.questions
  });

  return template
    .evaluate()
    .setTitle(meta.title || 'Тестирование')
    .addMetaTag('viewport', 'width=device-width, initial-scale=1')
    .setXFrameOptionsMode(HtmlService.XFrameOptionsMode.ALLOWALL);
}

function createForm(payload) {
  var form = FormApp.create(payload.title || ('Тест ' + payload.test_id));
  if (payload.description) {
    form.setDescription(payload.description);
  }
  form.setCollectEmail(false);
  form.setAllowResponseEdits(false);
  form.setProgressBar(true);

  var lastName = form.addTextItem().setTitle('Фамилия').setRequired(true);
  var firstName = form.addTextItem().setTitle('Имя').setRequired(true);
  var middleName = form.addTextItem().setTitle('Отчество').setRequired(false);
  var phone = form.addTextItem().setTitle('Номер телефона').setRequired(true);

  var branches = payload.branches || [];

  var branchLabels = sanitizeChoices_(branches.map(function (b) { return b.name; }));
  var branchItem = form.addListItem().setTitle('Филиал').setRequired(true);
  if (branchLabels.length) {
    branchItem.setChoiceValues(branchLabels);
  }

  form.addPageBreakItem().setTitle('Вопросы теста');

  var itemMap = {};
  var questions = [];
  (payload.questions || []).forEach(function (q, index) {
    var title = safeTitle_(q.content, index + 1);

    var options = sanitizeChoices_(q.options);
    var item;
    if (q.type === 1 && options.length >= 2) {
      item = form.addMultipleChoiceItem().setTitle(title);
      item.setChoiceValues(options);
    } else if (q.type === 3) {
      item = form.addScaleItem().setTitle(title).setBounds(1, 10);
    } else {

      if (q.type === 1) {
        console.warn('вопрос ' + q.question_id + ': вариантов меньше двух, поле стало текстовым');
      }
      item = form.addParagraphTextItem().setTitle(title);
      options = [];
    }
    item.setRequired(false);

    var itemId = String(item.getId());
    itemMap[itemId] = q.question_id;
    questions.push({
      item_id: itemId,
      question_id: q.question_id,
      content: title,
      type: q.type,
      options: options
    });
  });

  var meta = {
    form_id: form.getId(),
    test_id: payload.test_id,
    title: payload.title,
    description: payload.description || '',
    duration_minutes: payload.duration_minutes,
    branches: branches,
    questions: questions,
    item_map: itemMap,
    identity: {
      last_name: String(lastName.getId()),
      first_name: String(firstName.getId()),
      middle_name: String(middleName.getId()),
      phone: String(phone.getId()),
      branch: String(branchItem.getId())
    },
    webhook_url: payload.webhook_url || '',
    last_sync: 0,
    created_at: new Date().toISOString()
  };
  writeForm_(meta);

  return {
    form_id: form.getId(),
    form_url: form.getPublishedUrl(),
    respondent_url: webAppUrl_() + '?form=' + form.getId(),
    edit_url: form.getEditUrl(),
    item_map: itemMap
  };
}

function closeForm(formId) {
  var lastErr = null;

  for (var attempt = 1; attempt <= 3; attempt++) {
    try {
      var form = FormApp.openById(formId);
      if (form.isAcceptingResponses()) {
        form.setAcceptingResponses(false);
      }
      return { form_id: formId, closed: true };
    } catch (err) {
      lastErr = err;
      var text = String(err && err.message ? err.message : err);
      if (/не найден|not found|no item with the given id|нет элемента/i.test(text)) {

        console.warn('форма ' + formId + ' не найдена в Drive, считаем закрытой');
        return { form_id: formId, closed: true, missing: true };
      }
      console.warn('попытка ' + attempt + ' закрыть форму ' + formId + ': ' + text);
      if (attempt < 3) {
        Utilities.sleep(1000 * attempt);
      }
    }
  }

  throw lastErr;
}

function submitAnswers(payload) {
  var meta = readForm_(payload.formId);
  if (!meta) {
    throw new Error('Ссылка недействительна');
  }

  var form = FormApp.openById(payload.formId);
  if (!form.isAcceptingResponses()) {
    throw new Error('Тест закрыт и больше не принимает ответы');
  }

  var itemsById = {};
  form.getItems().forEach(function (item) {
    itemsById[String(item.getId())] = item;
  });

  var response = form.createResponse();

  addIdentityResponse_(response, itemsById, meta.identity.last_name, payload.lastName);
  addIdentityResponse_(response, itemsById, meta.identity.first_name, payload.firstName);
  addIdentityResponse_(response, itemsById, meta.identity.middle_name, payload.middleName);
  addIdentityResponse_(response, itemsById, meta.identity.phone, payload.phone);
  addIdentityResponse_(response, itemsById, meta.identity.branch, payload.branchLabel);

  (payload.answers || []).forEach(function (a) {
    var item = itemsById[a.itemId];
    if (!item || a.answer === '' || a.answer === null || a.answer === undefined) {
      return;
    }

    try {
      var ir = buildItemResponse_(item, a.answer);
      if (ir) {
        response.withItemResponse(ir);
      }
    } catch (err) {
      console.warn('ответ на вопрос ' + a.itemId + ' не записан в форму: ' + err);
    }
  });

  var submitted = response.submit();

  var body = buildSubmission_(meta, submitted, payload);
  var delivered = delivered_(sendToBackend_(body));

  return { ok: true, delivered: delivered, response_id: body.response_id };
}

function buildSubmission_(meta, formResponse, payload) {
  var branch = parseBranch_(payload.branchLabel, meta.branches);
  var answers = [];

  (payload.answers || []).forEach(function (a) {
    answers.push({
      item_id: a.itemId,
      question_id: meta.item_map[a.itemId] || 0,
      answer: a.answer === null || a.answer === undefined ? '' : String(a.answer)
    });
  });

  return {
    form_id: meta.form_id,
    response_id: formResponse.getId(),
    last_name: payload.lastName || '',
    first_name: payload.firstName || '',
    middle_name: payload.middleName || '',
    phone: payload.phone || '',
    branch_code: branch.code,
    branch_name: branch.name,
    auto_submitted: !!payload.autoSubmitted,
    submitted_at: new Date().toISOString(),
    answers: answers
  };
}

function setupSync() {
  ScriptApp.getProjectTriggers().forEach(function (t) {
    if (t.getHandlerFunction() === 'syncResponses') {
      ScriptApp.deleteTrigger(t);
    }
  });
  ScriptApp.newTrigger('syncResponses').timeBased().everyMinutes(SYNC_INTERVAL_MINUTES).create();
  return 'сверка включена: раз в ' + SYNC_INTERVAL_MINUTES + ' мин.';
}

function syncResponses() {
  listForms_().forEach(function (formId) {
    var meta = readForm_(formId);
    if (!meta) {
      return;
    }

    if (meta.unknown_to_backend) {
      return;
    }

    var form;
    try {
      form = FormApp.openById(formId);
    } catch (err) {
      return;
    }

    var since = meta.last_sync ? new Date(meta.last_sync) : null;
    var responses = since ? form.getResponses(since) : form.getResponses();
    var newest = meta.last_sync || 0;
    var unknown = 0;

    responses.forEach(function (r) {
      var body = submissionFromFormResponse_(meta, r);
      if (!body) {
        return;
      }
      var result = sendToBackend_(body);
      if (delivered_(result)) {
        var ts = r.getTimestamp().getTime();
        if (ts > newest) {
          newest = ts;
        }
        return;
      }
      if (result && result.rejected && (result.code === 404 || result.code === 401)) {
        unknown++;
      }
    });

    meta.last_sync = newest;
    if (unknown) {
      meta.unknown_strikes = (meta.unknown_strikes || 0) + 1;
      if (meta.unknown_strikes >= 3) {
        meta.unknown_to_backend = true;
        console.warn('форма ' + formId + ' (' + meta.title + ') бэкенду неизвестна ' +
          '(404/401 три сверки подряд) — исключена из сверки. ' +
          'Проверьте базу и секрет, затем запустите resendAll().');
      }
    } else {
      meta.unknown_strikes = 0;
    }
    writeForm_(meta);
  });
}

function submissionFromFormResponse_(meta, formResponse) {
  var byItem = {};
  formResponse.getItemResponses().forEach(function (ir) {
    byItem[String(ir.getItem().getId())] = ir.getResponse();
  });

  var branch = parseBranch_(byItem[meta.identity.branch] || '', meta.branches);
  var answers = [];
  Object.keys(meta.item_map).forEach(function (itemId) {
    var value = byItem[itemId];
    answers.push({
      item_id: itemId,
      question_id: meta.item_map[itemId],
      answer: value === undefined || value === null ? '' : String(value)
    });
  });

  var phone = byItem[meta.identity.phone] || '';
  if (!phone) {

    return null;
  }

  return {
    form_id: meta.form_id,
    response_id: formResponse.getId(),
    last_name: byItem[meta.identity.last_name] || '',
    first_name: byItem[meta.identity.first_name] || '',
    middle_name: byItem[meta.identity.middle_name] || '',
    phone: String(phone),
    branch_code: branch.code,
    branch_name: branch.name,
    auto_submitted: false,
    submitted_at: formResponse.getTimestamp().toISOString(),
    answers: answers
  };
}

function sendToBackend_(body) {
  var props = PropertiesService.getScriptProperties();
  var meta = body.form_id ? readForm_(body.form_id) : null;
  var url = (meta && meta.webhook_url) || props.getProperty(PROP_WEBHOOK);
  var secret = props.getProperty(PROP_SECRET);
  if (!url || !secret) {
    return false;
  }

  var options = {
    method: 'post',
    contentType: 'application/json',
    headers: {
      'X-Gform-Secret': secret,

      'ngrok-skip-browser-warning': 'true'
    },
    payload: JSON.stringify(body),
    muteHttpExceptions: true
  };

  for (var attempt = 1; attempt <= 3; attempt++) {
    try {
      var res = UrlFetchApp.fetch(url, options);
      var code = res.getResponseCode();
      var text = res.getContentText();

      var parsed = null;
      try {
        parsed = JSON.parse(text);
      } catch (e) {
        parsed = null;
      }

      if (!parsed) {

        console.warn('ответ дошёл не до бэкенда, а до чего-то ещё. URL: ' + url +
          ' | код ' + code + ' | тело: ' + text.slice(0, 200));
      } else if (code >= 200 && code < 300 && parsed.attempt_id) {
        return true;
      } else if (code === 400 || code === 401 || code === 404) {

        console.warn('бэкенд отклонил ответ: ' + code + ' ' + text.slice(0, 200));
        return { rejected: true, code: code, error: String(parsed.error || '') };
      } else {
        console.warn('попытка ' + attempt + ': бэкенд ответил ' + code);
      }
    } catch (err) {
      console.warn('попытка ' + attempt + ' не удалась: ' + err);
    }
    if (attempt < 3) {
      Utilities.sleep(1000 * attempt);
    }
  }
  return false;
}

function delivered_(result) {
  return result === true;
}

function resendAll() {
  var lines = [];
  listForms_().forEach(function (formId) {
    var meta = readForm_(formId);
    if (!meta) {
      return;
    }
    meta.last_sync = 0;

    meta.unknown_to_backend = false;
    meta.unknown_strikes = 0;
    writeForm_(meta);

    var form;
    try {
      form = FormApp.openById(formId);
    } catch (err) {
      lines.push(formId + ': форма недоступна');
      return;
    }

    var responses = form.getResponses();
    var sent = 0;
    responses.forEach(function (r) {
      var payload = submissionFromFormResponse_(meta, r);
      if (payload && delivered_(sendToBackend_(payload))) {
        sent++;
      }
    });
    lines.push(meta.title + ' (' + formId + '): ответов в форме ' + responses.length + ', доставлено ' + sent);
  });

  var report = lines.length ? lines.join('\n') : 'форм пока нет';
  console.log(report);
  return report;
}

function checkSetup() {
  var props = PropertiesService.getScriptProperties();
  var secret = props.getProperty(PROP_SECRET);
  var webhook = props.getProperty(PROP_WEBHOOK);
  var lines = [];

  lines.push(secret ? '✓ SHARED_SECRET задан' : '✗ SHARED_SECRET не задан');
  lines.push(webhook ? '✓ BACKEND_WEBHOOK_URL: ' + webhook : '✗ BACKEND_WEBHOOK_URL не задан');

  if (secret && webhook) {
    var pingUrl = webhook.replace(/\/submission$/, '/ping');
    try {
      var res = UrlFetchApp.fetch(pingUrl, {
        method: 'post',
        contentType: 'application/json',
        headers: {
          'X-Gform-Secret': secret,
          'ngrok-skip-browser-warning': 'true'
        },
        payload: '{}',
        muteHttpExceptions: true
      });
      lines.push(res.getResponseCode() === 200
        ? '✓ бэкенд отвечает, секрет верный'
        : '✗ бэкенд ответил ' + res.getResponseCode() + ': ' + res.getContentText());
    } catch (err) {
      lines.push('✗ бэкенд недоступен: ' + err);
    }
  }

  lines.push('web app: ' + webAppUrl_());
  lines.push('форм создано: ' + listForms_().length);

  var report = lines.join('\n');
  console.log(report);
  return report;
}

function json_(payload) {
  return ContentService
    .createTextOutput(JSON.stringify(payload))
    .setMimeType(ContentService.MimeType.JSON);
}

var MAX_CHOICE_LENGTH = 200;
var MAX_TITLE_LENGTH = 500;

function sanitizeChoices_(values) {
  var seen = {};
  var out = [];

  (values || []).forEach(function (value) {
    var text = String(value === null || value === undefined ? '' : value)
      .replace(/\s+/g, ' ')
      .trim();
    if (!text) {
      return;
    }
    if (text.length > MAX_CHOICE_LENGTH) {
      text = text.slice(0, MAX_CHOICE_LENGTH - 1) + '…';
    }
    var key = text.toLowerCase();
    if (seen[key]) {
      console.warn('повторяющийся вариант ответа отброшен: ' + text);
      return;
    }
    seen[key] = true;
    out.push(text);
  });

  return out;
}

function safeTitle_(content, number) {
  var text = String(content === null || content === undefined ? '' : content).trim();
  if (!text) {
    return 'Вопрос ' + number;
  }
  if (text.length > MAX_TITLE_LENGTH) {
    return text.slice(0, MAX_TITLE_LENGTH - 1) + '…';
  }
  return text;
}

function requireSecret_(given) {
  var expected = PropertiesService.getScriptProperties().getProperty(PROP_SECRET);
  if (!expected || String(given) !== String(expected)) {
    throw new Error('неверный секрет');
  }
}

function webAppUrl_() {
  return ScriptApp.getService().getUrl();
}

function addIdentityResponse_(response, itemsById, itemId, value) {
  if (!itemId || value === undefined || value === null || value === '') {
    return;
  }
  var item = itemsById[itemId];
  if (!item) {
    return;
  }

  try {
    response.withItemResponse(textOrList_(item, value));
  } catch (err) {
    console.warn('поле ' + itemId + ' не записано в форму: ' + err);
  }
}

function textOrList_(item, value) {
  if (item.getType() === FormApp.ItemType.LIST) {
    return item.asListItem().createResponse(String(value));
  }
  return item.asTextItem().createResponse(String(value));
}

function buildItemResponse_(item, value) {
  switch (item.getType()) {
    case FormApp.ItemType.MULTIPLE_CHOICE:
      return item.asMultipleChoiceItem().createResponse(String(value));
    case FormApp.ItemType.SCALE:

      var scale = Number(String(value).replace(',', '.'));
      if (!isFinite(scale)) {
        console.warn('ответ на шкалу не число, пропущен: ' + value);
        return null;
      }
      return item.asScaleItem().createResponse(Math.round(scale));
    case FormApp.ItemType.PARAGRAPH_TEXT:
      return item.asParagraphTextItem().createResponse(String(value));
    case FormApp.ItemType.TEXT:
      return item.asTextItem().createResponse(String(value));
    case FormApp.ItemType.LIST:
      return item.asListItem().createResponse(String(value));
    default:
      return null;
  }
}

function parseBranch_(label, branches) {
  var raw = String(label || '').trim();
  if (!raw) {
    return { code: '', name: '' };
  }
  var known = (branches || []).filter(function (b) { return String(b.name).trim() === raw; })[0];
  if (known) {
    return { code: String(known.code), name: String(known.name) };
  }
  var parts = raw.split('—');
  if (parts.length < 2) {
    return { code: raw, name: raw };
  }
  return { code: parts[0].trim(), name: parts.slice(1).join('—').trim() };
}

function errorPage_(title, text) {
  var html = '<!doctype html><meta charset="utf-8">' +
    '<meta name="viewport" content="width=device-width, initial-scale=1">' +
    '<div style="font:16px/1.5 system-ui,Segoe UI,Roboto,sans-serif;max-width:520px;' +
    'margin:15vh auto;padding:32px;text-align:center;color:#1f1f1f;border-top:3px solid #e31e24">' +
    '<div style="font-size:40px;margin-bottom:12px">🔒</div>' +
    '<h1 style="font-size:22px;margin:0 0 8px">' + title + '</h1>' +
    '<p style="margin:0;color:#5b6b64">' + text + '</p></div>';
  return HtmlService.createHtmlOutput(html).setTitle(title);
}

function listForms_() {
  var raw = PropertiesService.getScriptProperties().getProperty(PROP_FORMS);
  return raw ? JSON.parse(raw) : [];
}

function readForm_(formId) {
  var raw = PropertiesService.getScriptProperties().getProperty(FORM_PREFIX + formId);
  return raw ? JSON.parse(raw) : null;
}

function writeForm_(meta) {
  var props = PropertiesService.getScriptProperties();
  props.setProperty(FORM_PREFIX + meta.form_id, JSON.stringify(meta));

  var forms = listForms_();
  if (forms.indexOf(meta.form_id) === -1) {
    forms.push(meta.form_id);
    props.setProperty(PROP_FORMS, JSON.stringify(forms));
  }
}
