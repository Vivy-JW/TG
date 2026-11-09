using System;
using System.Collections.Generic;
using System.Linq;
using System.Text.RegularExpressions;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.UndoRedoCommands;
using SpecDesignerCommon.ViewModel;
using UndoRedoFramework.Commands;

namespace SpecDesignerCommon.UndoRedo
{
	// Token: 0x02000006 RID: 6
	public class PasteComponentUndoRedoCommand : IUndoRedoCommand
	{
		// Token: 0x0600003A RID: 58 RVA: 0x00002D18 File Offset: 0x00000F18
		public PasteComponentUndoRedoCommand(DesignerClipboardData data, XmlElement container)
		{
			this._container = container;
			this._key = this._container.Key;
			this.GetClipboradOjbect(data);
		}

		// Token: 0x0600003B RID: 59 RVA: 0x00002D46 File Offset: 0x00000F46
		public PasteComponentUndoRedoCommand(DesignerClipboardData data, XmlElement container, int index)
			: this(data, container)
		{
			this._index = index;
		}

		// Token: 0x0600003C RID: 60 RVA: 0x00002D58 File Offset: 0x00000F58
		private bool ExistInContainer(string name)
		{
			foreach (XmlElement xmlElement in this._container.Nodes)
			{
				if (xmlElement.Name == name)
				{
					return true;
				}
			}
			return false;
		}

		// Token: 0x0600003D RID: 61 RVA: 0x00002DB8 File Offset: 0x00000FB8
		private void GetClipboradOjbect(DesignerClipboardData data)
		{
			int num = 0;
			int num2 = 1;
			XmlElement.GetMaxTabIndex(this.GetSpecificationInfo().FormNode, ref num);
			if (this._pasteComponents == null)
			{
				this._pasteComponents = new List<FormSpecModel>();
			}
			else
			{
				this._pasteComponents.Clear();
			}
			foreach (FormSpecModel formSpecModel in data.GetSelections(this._key))
			{
				if (ComponentFactory.IsIncludeProperties(formSpecModel.GeneroComponent.NodeName, "tabIndex"))
				{
					formSpecModel.GeneroComponent.TabIndex = num + num2;
					num2++;
				}
				if (this.GetSpecificationInfo().IsExists(formSpecModel.Name))
				{
					string text = formSpecModel.GeneroComponent.GetAttribute("tag");
					string text2 = "\\bcantdel\\b";
					Regex regex = new Regex(text2, RegexOptions.IgnoreCase);
					if (text != null && Regex.IsMatch(text, "\\bcantdel\\b", RegexOptions.IgnoreCase))
					{
						text = regex.Replace(text, "");
						text = text.Replace("  ", " ");
						formSpecModel.GeneroComponent.SetAttribute("tag", text);
					}
					if (this.ExistInContainer(formSpecModel.Name))
					{
						formSpecModel.GeneroComponent.GridX++;
					}
					string newName = ComponentFactory.GetNewName(formSpecModel.Name, this._key);
					formSpecModel.Name = newName;
					SpecNodeType specNodeType = formSpecModel.SpecNodeType;
					if (specNodeType == SpecNodeType.ACTION && formSpecModel.SpecAction == null)
					{
						formSpecModel.SpecAction = SpecActionNode.Create(this.GetSpecificationInfo(), formSpecModel.Name);
					}
				}
				this._pasteComponents.Add(formSpecModel);
			}
		}

		// Token: 0x0600003E RID: 62 RVA: 0x00002F68 File Offset: 0x00001168
		public void Undo()
		{
			if (ComponentHelper.Get(this._key).SelectedObjects.Contains(this._pasteComponents[0].GeneroComponent))
			{
				ComponentHelper.Get(this._key).ClearSelection();
			}
			foreach (FormSpecModel formSpecModel in this._pasteComponents)
			{
				this._container.RemoveNode(formSpecModel.GeneroComponent);
				this.GetSpecificationInfo().Remove(formSpecModel.Name);
			}
		}

		// Token: 0x0600003F RID: 63 RVA: 0x00003010 File Offset: 0x00001210
		public void Execute()
		{
			FormSizeComplexUndoRedoCommand formSizeComplexUndoRedoCommand = new FormSizeComplexUndoRedoCommand(this);
			SettingManager.Get().GetUndoRedoManager(this._container.Key).StartGroup(formSizeComplexUndoRedoCommand);
			int num = 0;
			int num2 = 0;
			int num3 = 0;
			int num4 = 0;
			bool flag = false;
			bool flag2 = false;
			for (int i = 0; i < this._pasteComponents.Count; i++)
			{
				FormSpecModel formSpecModel = this._pasteComponents[i];
				XmlElement generoComponent = formSpecModel.GeneroComponent;
				if (i == 0)
				{
					num3 = generoComponent.GridX;
					num4 = generoComponent.GridY;
				}
				else
				{
					flag = generoComponent.GridX > num3;
					flag2 = generoComponent.GridY > num4;
				}
				this._container.AddNodeAt(generoComponent, (-1 == this._index) ? this._container.Nodes.Count : (this._index + i));
				if (i == 0)
				{
					num = generoComponent.GridX - num3;
					num = ((num > 0) ? num : 0);
					num2 = generoComponent.GridY - num4;
					num2 = ((num2 > 0) ? num2 : 0);
				}
				generoComponent.GridY += (flag2 ? num2 : 0);
				generoComponent.GridX += (flag ? num : 0);
				this.GetSpecificationInfo().Add(formSpecModel);
			}
			List<XmlElement> list = new List<XmlElement>();
			foreach (FormSpecModel formSpecModel2 in this._pasteComponents)
			{
				list.Add(formSpecModel2.GeneroComponent);
			}
			ComponentHelper.Get(this._key).MultipleSelection(list);
			SettingManager.Get().GetUndoRedoManager(this._container.Key).EndGroup(formSizeComplexUndoRedoCommand);
		}

		// Token: 0x06000040 RID: 64 RVA: 0x000031D0 File Offset: 0x000013D0
		public void Clear()
		{
			this._pasteComponents.Clear();
		}

		// Token: 0x06000041 RID: 65 RVA: 0x000031DD File Offset: 0x000013DD
		private SpecificationInfo GetSpecificationInfo()
		{
			return SettingManager.Get().GetTzpManger(this._key).SpecificationInfo;
		}

		// Token: 0x04000015 RID: 21
		private List<FormSpecModel> _pasteComponents;

		// Token: 0x04000016 RID: 22
		private XmlElement _container;

		// Token: 0x04000017 RID: 23
		private PackageKey _key;

		// Token: 0x04000018 RID: 24
		private int _index = -1;
	}
}
