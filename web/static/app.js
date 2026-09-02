(function () {
	"use strict";

	var currentPath = window.__CURRENT_PATH__ || "";

	// Build breadcrumb links from currentPath.
	function renderBreadcrumb() {
		var bc = document.getElementById("breadcrumb");
		if (!bc) return;
		bc.innerHTML = "";
		var parts = currentPath.split("/").filter(Boolean);
		var acc = "";
		var a = document.createElement("a");
		a.href = "/?path=";
		a.textContent = "/";
		bc.appendChild(a);
		parts.forEach(function (p) {
			acc += "/" + p;
			bc.appendChild(document.createTextNode(" / "));
			var link = document.createElement("a");
			link.href = "/?path=" + encodeURIComponent(acc);
			link.textContent = p;
			bc.appendChild(link);
		});
	}
	renderBreadcrumb();

	// Upload via FormData.
	function uploadFiles(files) {
		if (!files || !files.length) return;
		var fd = new FormData();
		for (var i = 0; i < files.length; i++) fd.append("files[]", files[i]);
		var url = "/api/upload?path=" + encodeURIComponent(currentPath);
		fetch(url, { method: "POST", body: fd }).then(function (r) {
			if (r.ok) location.reload();
			else r.text().then(function (t) { alert("upload failed: " + t); });
		}).catch(function (e) { alert("upload error: " + e); });
	}

	// File picker.
	var fileInput = document.getElementById("fileInput");
	if (fileInput) {
		fileInput.addEventListener("change", function () {
			uploadFiles(fileInput.files);
			fileInput.value = "";
		});
	}

	// Drag & drop on the dropzone.
	var dz = document.getElementById("dropzone");
	["dragenter", "dragover"].forEach(function (ev) {
		dz.addEventListener(ev, function (e) {
			e.preventDefault();
			dz.classList.add("dragover");
		});
	});
	["dragleave", "drop"].forEach(function (ev) {
		dz.addEventListener(ev, function (e) {
			e.preventDefault();
			dz.classList.remove("dragover");
		});
	});
	dz.addEventListener("drop", function (e) {
		if (e.dataTransfer && e.dataTransfer.files) uploadFiles(e.dataTransfer.files);
	});

	// New file.
	var newFileBtn = document.getElementById("newFileBtn");
	if (newFileBtn) {
		newFileBtn.addEventListener("click", function () {
			var name = prompt("File name");
			if (!name) return;
			var url = "/api/newfile?path=" + encodeURIComponent(currentPath) + "&name=" + encodeURIComponent(name);
			fetch(url, { method: "POST" }).then(function (r) {
				if (r.ok) location.reload();
				else r.text().then(function (t) { alert("create file failed: " + t); });
			});
		});
	}

	// New folder.
	var newFolderBtn = document.getElementById("newFolderBtn");
	if (newFolderBtn) {
		newFolderBtn.addEventListener("click", function () {
			var name = prompt("Folder name");
			if (!name) return;
			var url = "/api/mkdir?path=" + encodeURIComponent(currentPath) + "&name=" + encodeURIComponent(name);
			fetch(url, { method: "POST" }).then(function (r) {
				if (r.ok) location.reload();
				else r.text().then(function (t) { alert("mkdir failed: " + t); });
			});
		});
	}

	// Row action buttons (event delegation).
	document.addEventListener("click", function (e) {
		var btn = e.target.closest("button[data-action]");
		if (!btn) return;
		var action = btn.getAttribute("data-action");
		var path = btn.getAttribute("data-path");
		var name = btn.getAttribute("data-name");
		var url = "/api/" + action + "?path=" + encodeURIComponent(path);

		if (action === "download") {
			window.location = url;
			return;
		}
		if (action === "blob") {
			fetch(url).then(function (r) {
				if (!r.ok) throw new Error("status " + r.status);
				return r.blob();
			}).then(function (blob) {
				var u = URL.createObjectURL(blob);
				var a = document.createElement("a");
				a.href = u;
				a.download = name;
				document.body.appendChild(a);
				a.click();
				a.remove();
				setTimeout(function () { URL.revokeObjectURL(u); }, 1000);
			}).catch(function (err) { alert("blob download failed: " + err); });
			return;
		}
		if (action === "delete") {
			if (!confirm("Delete " + name + "?")) return;
			fetch(url, { method: "POST" }).then(function (r) {
				if (r.ok) location.reload();
				else r.text().then(function (t) { alert("delete failed: " + t); });
			});
			return;
		}
		if (action === "edit") {
			openEditor(path, name);
			return;
		}
	});

	// ---- Text editor modal ----
	var modal = document.getElementById("editModal");
	var editArea = document.getElementById("editArea");
	var editTitle = document.getElementById("editTitle");
	var editPath = null;
	var saving = false;

	function openEditor(path, name) {
		editPath = path;
		editTitle.textContent = "Edit: " + name;
		editArea.value = "Loading...";
		modal.hidden = false;
		modal.classList.add("open");
		editArea.disabled = true;
		fetch("/api/read?path=" + encodeURIComponent(path))
			.then(function (r) {
				if (!r.ok) throw new Error("status " + r.status);
				return r.text();
			})
			.then(function (text) {
				editArea.value = text;
				editArea.disabled = false;
				editArea.focus();
			})
			.catch(function (err) {
				closeEditor();
				alert("read failed: " + err);
			});
	}

	function saveAndClose() {
		if (!editPath) return;
		if (saving) return;
		saving = true;
		var body = editArea.value;
		fetch("/api/save?path=" + encodeURIComponent(editPath), {
			method: "POST",
			headers: { "Content-Type": "text/plain; charset=utf-8" },
			body: body
		}).then(function (r) {
			saving = false;
			if (r.ok) {
				editPath = null;
				modal.classList.remove("open");
				modal.hidden = true;
				location.reload();
			} else {
				r.text().then(function (t) { alert("save failed: " + t); });
			}
		}).catch(function (err) {
			saving = false;
			alert("save error: " + err);
		});
	}

	// Save without closing (Ctrl+S).
	function saveOnly() {
		if (!editPath || saving) return;
		saving = true;
		fetch("/api/save?path=" + encodeURIComponent(editPath), {
			method: "POST",
			headers: { "Content-Type": "text/plain; charset=utf-8" },
			body: editArea.value
		}).then(function (r) {
			saving = false;
			if (!r.ok) r.text().then(function (t) { alert("save failed: " + t); });
		}).catch(function (err) {
			saving = false;
			alert("save error: " + err);
		});
	}

	function closeEditor() {
		if (saving) return;
		if (editPath) {
			saveAndClose();
		} else {
			modal.classList.remove("open");
			modal.hidden = true;
		}
	}

	if (modal) {
		// Click on backdrop (not inside the modal card) -> save & close.
		modal.addEventListener("click", function (e) {
			if (e.target === modal) closeEditor();
		});
	}

	document.addEventListener("keydown", function (e) {
		if (!modal || modal.hidden) return;
		if (e.key === "Escape") {
			e.preventDefault();
			closeEditor();
		} else if ((e.ctrlKey || e.metaKey) && (e.key === "s" || e.key === "S")) {
			e.preventDefault();
			saveOnly();
		}
	});
})();
