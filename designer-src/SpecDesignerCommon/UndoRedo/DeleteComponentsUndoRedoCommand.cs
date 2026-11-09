using System;
using System.Collections.Generic;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.UndoRedoCommands;
using SpecDesignerCommon.ViewModel;
using UndoRedoFramework.Commands;

namespace SpecDesignerCommon.UndoRedo
{
	// Token: 0x0200002E RID: 46
	public class DeleteComponentsUndoRedoCommand : AbstractComplexTriggerUndoRedoCommand
	{
		// Token: 0x06000161 RID: 353 RVA: 0x00007430 File Offset: 0x00005630
		public DeleteComponentsUndoRedoCommand(XmlElement parent, List<XmlElement> list)
		{
			ComponentType type = parent.Type;
			if (type != ComponentType.Folder)
			{
				switch (type)
				{
				case ComponentType.Table:
				case ComponentType.Tree:
					break;
				default:
					goto IL_004C;
				}
			}
			if (list.Count == parent.Nodes.Count)
			{
				list.Clear();
				list.Add(parent);
				parent = parent.Parent;
			}
			IL_004C:
			this._parent = parent;
			this._key = this._parent.Key;
			this._list = new List<XmlElement>();
			this._specList = new Dictionary<XmlElement, DeleteComponentsUndoRedoCommand.DeleteRecord>();
			foreach (XmlElement xmlElement in list)
			{
				this.GetSpecContainsChildren(xmlElement);
				this._list.Add(xmlElement);
			}
		}

		// Token: 0x06000162 RID: 354 RVA: 0x00007504 File Offset: 0x00005704
		private void GetSpecContainsChildren(XmlElement children)
		{
			for (int i = 0; i < children.Nodes.Count; i++)
			{
				XmlElement xmlElement = children.Nodes[i];
				this.GetSpecContainsChildren(xmlElement);
			}
			FormSpecModel formSpecModel = this.GetSpecificationInfo().FindNodeByName(children.Name);
			DeleteComponentsUndoRedoCommand.DeleteRecord deleteRecord = new DeleteComponentsUndoRedoCommand.DeleteRecord(children.Parent, children.Index, formSpecModel);
			if (!this._specList.ContainsKey(children))
			{
				this._specList.Add(children, deleteRecord);
			}
		}

		// Token: 0x06000163 RID: 355 RVA: 0x0000757C File Offset: 0x0000577C
		public override void Undo()
		{
			foreach (XmlElement xmlElement in this._list)
			{
				switch (this._parent.Type)
				{
				case ComponentType.HBox:
				case ComponentType.Table:
				case ComponentType.Tree:
				case ComponentType.VBox:
					this._parent.AddNodeAt(xmlElement, this._specList[xmlElement].Index);
					break;
				case ComponentType.Page:
				case ComponentType.RadioGroup:
				case ComponentType.ScrollGrid:
					goto IL_0067;
				default:
					goto IL_0067;
				}
				IL_0073:
				this.RecoverSpec(xmlElement);
				continue;
				IL_0067:
				this._parent.AddNode(xmlElement);
				goto IL_0073;
			}
			ComponentHelper.Get(this._key).MultipleSelection(this._list);
			this.GetSpecificationInfo().DatabaseSource.RefreshUsed(this.GetSpecificationInfo().FormSpeDictionary);
		}

		// Token: 0x06000164 RID: 356 RVA: 0x00007660 File Offset: 0x00005860
		public override void Execute()
		{
			FormSizeComplexUndoRedoCommand formSizeComplexUndoRedoCommand = new FormSizeComplexUndoRedoCommand(this);
			SettingManager.Get().GetUndoRedoManager(this._parent.Key).StartGroup(formSizeComplexUndoRedoCommand);
			ComponentHelper.Get(this._key).ClearSelection();
			foreach (XmlElement xmlElement in this._list)
			{
				this._parent.RemoveNode(xmlElement);
			}
			this.DeleteSpec();
			ComponentHelper.Get(this._key).AddSelection(this._parent, false);
			SettingManager.Get().GetUndoRedoManager(this._parent.Key).EndGroup(formSizeComplexUndoRedoCommand);
			this.GetSpecificationInfo().DatabaseSource.RefreshUsed(this.GetSpecificationInfo().FormSpeDictionary);
		}

		// Token: 0x06000165 RID: 357 RVA: 0x00007740 File Offset: 0x00005940
		private void DeleteSpec()
		{
			foreach (KeyValuePair<XmlElement, DeleteComponentsUndoRedoCommand.DeleteRecord> keyValuePair in this._specList)
			{
				this.GetSpecificationInfo().Remove(keyValuePair.Key.Name);
			}
		}

		// Token: 0x06000166 RID: 358 RVA: 0x000077A4 File Offset: 0x000059A4
		private void RecoverSpec(XmlElement element)
		{
			foreach (XmlElement xmlElement in element.Nodes)
			{
				this.RecoverSpec(xmlElement);
			}
			DeleteComponentsUndoRedoCommand.DeleteRecord deleteRecord = this._specList[element];
			FormSpecModel specModel = deleteRecord.SpecModel;
			this.SetSpecStatusAsUpdate(specModel.SpecAction);
			this.SetSpecStatusAsUpdate(specModel.SpecField);
			this.SetSpecStatusAsUpdate(specModel.SpecHelpCode);
			this.SetSpecStatusAsUpdate(specModel.SpecMultiLang);
			this.SetSpecStatusAsUpdate(specModel.SpecProgRel);
			this.SetSpecStatusAsUpdate(specModel.SpecReference);
			this.SetSpecStatusAsUpdate(specModel.SpecTree);
			this.GetSpecificationInfo().FormSpeDictionary.Add(element.Name, specModel);
		}

		// Token: 0x06000167 RID: 359 RVA: 0x00007870 File Offset: 0x00005A70
		private void SetSpecStatusAsUpdate(AbstractSpecNode node)
		{
			if (node == null)
			{
				return;
			}
			node.Status &= ~SpecStatus.DELETE;
			node.Status |= SpecStatus.MODIFY;
		}

		// Token: 0x06000168 RID: 360 RVA: 0x00007893 File Offset: 0x00005A93
		public override void Clear()
		{
			if (this._specList != null)
			{
				this._specList.Clear();
			}
		}

		// Token: 0x06000169 RID: 361 RVA: 0x000078A8 File Offset: 0x00005AA8
		private SpecificationInfo GetSpecificationInfo()
		{
			if (this._parent != null)
			{
				return SettingManager.Get().GetTzpManger(this._parent.Key).SpecificationInfo;
			}
			return null;
		}

		// Token: 0x0400008D RID: 141
		private Dictionary<XmlElement, DeleteComponentsUndoRedoCommand.DeleteRecord> _specList;

		// Token: 0x0400008E RID: 142
		private List<XmlElement> _list;

		// Token: 0x0400008F RID: 143
		private XmlElement _parent;

		// Token: 0x04000090 RID: 144
		private PackageKey _key;

		// Token: 0x0200002F RID: 47
		private class DeleteRecord
		{
			// Token: 0x17000055 RID: 85
			// (get) Token: 0x0600016A RID: 362 RVA: 0x000078CE File Offset: 0x00005ACE
			// (set) Token: 0x0600016B RID: 363 RVA: 0x000078D6 File Offset: 0x00005AD6
			public XmlElement Parent { get; set; }

			// Token: 0x17000056 RID: 86
			// (get) Token: 0x0600016C RID: 364 RVA: 0x000078DF File Offset: 0x00005ADF
			// (set) Token: 0x0600016D RID: 365 RVA: 0x000078E7 File Offset: 0x00005AE7
			public int Index { get; set; }

			// Token: 0x17000057 RID: 87
			// (get) Token: 0x0600016E RID: 366 RVA: 0x000078F0 File Offset: 0x00005AF0
			// (set) Token: 0x0600016F RID: 367 RVA: 0x000078F8 File Offset: 0x00005AF8
			public FormSpecModel SpecModel { get; set; }

			// Token: 0x06000170 RID: 368 RVA: 0x00007901 File Offset: 0x00005B01
			public DeleteRecord(XmlElement parent, int index, FormSpecModel spec)
			{
				this.Parent = parent;
				this.Index = index;
				this.SpecModel = spec;
			}
		}
	}
}
