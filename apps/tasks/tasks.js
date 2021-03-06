function main(ctxIn, page) {
  var sessData = {};
  var sessDeleted = {}
  var ctx = jscion(sessData, sessDeleted, ctxIn);
  page.statusChoices = page.statusChoices || findStatusChoices(ctx)
  page.search = page.search || "";
  page.tasks = page.tasks || [];
  page.taskCountsByStatus = page.taskCountsByStatus || {};
  page.obj = findTask(ctx, page.tasks, page.ident) || page.obj;

  if (!page.r || page.controller != page.prev.controller) {
    page.r = registerEventHandlers(ctx, page.render("tasks"));
  }

  page.r.update("tasks");
  page.r.update("tasksByStatus");
  renderTask(ctx, page.r, page.obj);
}

function registerEventHandlers(ctx, r) {
  r.on({
    "newTask": function(event) {
      var task = ctx.newObj("task", { "title": (event.node.value || "").trim() }).result;
      if (task.title) {
        r.get("tasks").unshift(task);
        updateTask(ctx, r, task, task);
        event.node.value = "";
        event.node.focus();
      }
    },
    "cloneTask": function() {
      var orig = r.get("obj");
      var task = ctx.newObj("task", { "title": orig.title,
                                      "description": orig.description }).result;
      r.get("tasks").unshift(task);
      updateTask(ctx, r, task, task);
    },
    "saveTask": function() {
      var edit = r.get("objEdit");
      var errs = ctx.validateObj(edit);
      if (_.some(_.values(errs), _.isString)) {
        r.set("objEditErrs", errs);
        return;
      }
      updateTask(ctx, r, findTask(ctx, r.get("tasks"), edit.ident), edit);
    },
    "deleteTask": function() {
      if (!confirm("are you sure you want to delete this task?")) {
        return;
      }
      var task = r.get("obj");
      r.set("tasks", _.reject(r.get("tasks"), function(t) { return t.ident == task.ident; }));
      r.set("ident", "app-info");
      updateTask(ctx, r, ctx.getObj("app-info").result || {});
    },
    "editTask": function() {
      renderTask(ctx, r, r.get("obj"), { "doEdit": !r.get("doEdit") });
      if (r.get("doEdit")) {
        setTimeout(function() { $("#objEdit_title").focus(); });
      }
    },
    "addMessage": function() {
      renderTask(ctx, r, r.get("obj"), { "doMessage": !r.get("doMessage") });
      if (r.get("doMessage")) {
        setTimeout(function() { $("#messageVal").focus(); });
      }
    },
    "saveMessage": function() {
      var msg = (r.get("messageVal") + "").trim();
      var task = r.get("obj");
      if (task && msg) {
        ctx.newChild(task, "messages", { "message": msg }).result;
      }
      updateTask(ctx, r, task, task);
    },
    "deleteMessage": function(event) {
      var task = r.get("obj");
      if (confirm("are you sure you want to delete that message?")) {
        task.messages.splice(parseInt(event.node.value), 1);
      }
      updateTask(ctx, r, task, task);
    },
    "changeTaskStatus": function(event) {
      var edit = r.get("objEdit");
      var newStatus =
        (_.findWhere((ctx.getObj("stateMachine-taskStatus").result || {}).transitions || [],
                     { "from": edit.status, "on": event.node.value }) || {}).to;
      if (newStatus) {
        edit.status = newStatus;
        updateTask(ctx, r, findTask(ctx, r.get("tasks"), edit.ident), edit,
                   ", to " + newStatus);
      }
    }
  });
  return r;
}

function findTask(ctx, tasks, ident) {
  return ctx.findObj(where).result || _.find(tasks, where);
  function where(task) { return task.ident == ident; };
}

function renderTask(ctx, r, task, extras) {
  var tasksByStatus = _.groupBy(r.get("tasks"), "status");
  r.set(_.defaults(extras || {}, {
        "obj": task,
        "objEdit": _.clone(task),
        "objEditErrs": null,
        "doEdit": false,
        "doMessage": false,
        "messageVal": "",
        "tasksByStatus": function(status) { return tasksByStatus[status]; },
        "stringContains": function(s, x) { return (s + "").indexOf(x + "") >= 0; }
      }));
  r.set("objJSON", JSON.stringify(task));
  r.set("taskStatusTransitions",
        task && _.filter((ctx.getObj("stateMachine-taskStatus").result || {}).transitions || [],
                         function(t) { return t.from == task.status; }));
}

function updateTask(ctx, r, orig, edit, msgSuffix) {
  edit = edit || orig;
  var changes = _.compact(_.map(_.keys(orig), function(k) { return (orig[k] != edit[k]) && k; }));
  if (changes.length > 0) {
    edit.updatedAt = new Date().toJSON();
    _.extend(orig, edit);
    _.each(_.keys(orig), function(k) {
        if (_.isString(orig[k])) { orig[k] = orig[k].trim(); }
      });
    var m = "(" + changes.join(", ") + " edited" + (msgSuffix || "") + ")";
    ctx.newChild(orig, "messages", { "message": m }).result;
  }
  renderTask(ctx, r, orig);
  r.update("tasks");
  updateServerAsync(ctx, r);
}

function findStatusChoices(ctx) {
  return _.findWhere(ctx.getClassByName("task").result.properties, { "name": "status" }).valueChoices;
}

function updateServerAsync(ctx, r) {
  r.set("syncingByIdent", _.indexBy(r.get("tasks"), "ident"));
}
