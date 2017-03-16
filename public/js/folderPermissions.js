chosenUsers = $('#slcUsers').chosen({
  no_results_text: "No users found"
});

let folderId = window.location.pathname.split('/').splice(-1)[0];
let btnAdd = document.getElementById('btnAdd');
let slcUsers = document.getElementById('slcUsers');
let scrUserPermissions = document.getElementById('scrUserPermissions');
let tbody = document.querySelector('#tblPermissions>tbody');
let frmFolderPermissions = document.querySelector('#frmFolderPermissions');
let hdnFolderPermissions = document.querySelector('#hdnFolderPermissions');
let tmpUserRow = document.getElementById('tmpUserRow');
let tr = tmpUserRow.content.querySelector('tr');

let userRow = {
  tr: tr,
  tdName: tr.querySelector('td'),
}

let userData = JSON.parse(scrUserPermissions.innerText)

btnAdd.addEventListener('click', evt => {
  let user = findUser(slcUsers.value);

  let title = `Level: ${user.level}`;
  if (user.admin) {
    title += "\nAdmin";
  }
  if (user.tech) {
    title += "\nTech"
  }

  userRow.tr.setAttribute('data-userid', user.id);
  userRow.tdName.setAttribute('title', title);
  userRow.tdName.innerText = user.name;
  
  let clone = document.importNode(tr, true);
  clone.querySelector('th>a').addEventListener('click', deleteRow);
  tbody.appendChild(clone);
  slcUsers.options[slcUsers.selectedIndex].remove()
  chosenUsers.trigger('chosen:updated');
});

frmFolderPermissions.addEventListener('submit', save, true);

function save(evt) {
  let rows = tbody.querySelectorAll('tr');
  let permissions = [];
  let permission = {};

  rows.forEach(element => {
    permission = {
      id: element.getAttribute('data-permissionid'),
      folderId: folderId,
      userId: element.getAttribute('data-userid'),
      list: element.querySelector('input[data-permission=list]').checked,
      read: element.querySelector('input[data-permission=read]').checked,
      write: element.querySelector('input[data-permission=write]').checked,
      create: element.querySelector('input[data-permission=create]').checked,
      delete: element.querySelector('input[data-permission=delete]').checked
    };
    permissions.push(permission);
  });

  hdnFolderPermissions.value = JSON.stringify(permissions);
}

function deleteRow(evt) {
  let row = evt.target.parentElement.parentElement;
  let user = findUser(row.getAttribute('data-userid'));

  let option = document.createElement('option');
  option.value = user.id;
  option.innerText = user.name;

  slcUsers.appendChild(option);
  chosenUsers.trigger('chosen:updated');

  row.remove();
}

function findUser(id) {
  return userData.users.find(user => user.id === id);
}