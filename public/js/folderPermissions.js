chosenUsers = $('#slcUsers').chosen({
  no_results_text: "No users found"
});

let btnAdd = document.getElementById('btnAdd');
let slcUsers = document.getElementById('slcUsers');
let scrUserPermissions = document.getElementById('scrUserPermissions');
let tbody = document.querySelector("#tblPermissions>tbody");

let tmpUserRow = document.getElementById('tmpUserRow');
let tr = tmpUserRow.content.querySelector('tr');

let userRow = {
  tr: tr,
  tdName: tr.querySelector('td'),
  cbxList: tr.querySelector('input[data-permission=list]'),
  cbxRead: tr.querySelector('input[data-permission=read]'),
  cbxWrite: tr.querySelector('input[data-permission=write]'),
  cbxCreate: tr.querySelector('input[data-permission=create]'),
  cbxDelete: tr.querySelector('input[data-permission=delete]')
}

let userData = JSON.parse(scrUserPermissions.innerText)

btnAdd.addEventListener("click", evt => {
  userIndex = 0;
  user = userData.users.find((index, user) => {
    userIndex = index;
    return user.id === slcUsers.value
  });

  title = `Level: ${user.level}`;
  if (user.admin) {
    title += "\nAdmin";
  }
  if (user.tech) {
    title += "\nTech"
  }

  userRow.tr.setAttribute('data-userid', user.id);
  userRow.tdName.setAttribute('title', title);
  userRow.tdName.innerText = user.name;
  
  clone = document.importNode(tr, true);
  tbody.appendChild(clone);
  slcUsers.options[slcUsers.selectedIndex].remove()
  chosenUsers.trigger('chosen:updated');
});