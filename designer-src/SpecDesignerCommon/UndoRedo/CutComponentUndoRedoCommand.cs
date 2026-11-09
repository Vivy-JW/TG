using System;
using System.Collections.Generic;
using System.Windows;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;
using UndoRedoFramework.Commands;

namespace SpecDesignerCommon.UndoRedo
{
	// Token: 0x02000057 RID: 87
	public class CutComponentUndoRedoCommand : IUndoRedoCommand
	{
		// Token: 0x060002EF RID: 751 RVA: 0x0000C718 File Offset: 0x0000A918
		public CutComponentUndoRedoCommand(XmlElement container, List<XmlElement> list)
		{
			this.isCut = false;
			this._container = container;
			this._key = this._container.Key;
			this._list = new Dictionary<FormSpecModel, int>();
			Clipboard.Clear();
			DesignerClipboardData designerClipboardData = new DesignerClipboardData(this._key, this._container.Type);
			SortPosition sortPosition = new SortPosition();
			list.Sort(sortPosition);
			foreach (XmlElement xmlElement in list)
			{
				designerClipboardData.AppendSelectedItem(xmlElement);
				FormSpecModel formSpecModel = this.GetSpecificationInfo().FindNodeByName(xmlElement.Name);
				this._list.Add(formSpecModel, xmlElement.Index);
			}
			IDataObject dataObject = new DataObject();
			dataObject.SetData(typeof(DesignerClipboardData).FullName, designerClipboardData);
			Clipboard.SetDataObject(designerClipboardData, false);
		}

		// Token: 0x060002F0 RID: 752 RVA: 0x0000C80C File Offset: 0x0000AA0C
		public void Undo()
		{
			this.isCut = false;
			foreach (KeyValuePair<FormSpecModel, int> keyValuePair in this._list)
			{
				this._container.AddNodeAt(keyValuePair.Key.GeneroComponent, keyValuePair.Value);
				this.GetSpecificationInfo().FormSpeDictionary.Add(keyValuePair.Key.Name, keyValuePair.Key);
			}
		}

		// Token: 0x060002F1 RID: 753 RVA: 0x0000C8A0 File Offset: 0x0000AAA0
		public void Execute()
		{
			ComponentHelper.Get(this._key).ClearSelection();
			foreach (KeyValuePair<FormSpecModel, int> keyValuePair in this._list)
			{
				this._container.RemoveNode(keyValuePair.Key.GeneroComponent);
				this.GetSpecificationInfo().Remove(keyValuePair.Key.Name);
			}
		}

		// Token: 0x060002F2 RID: 754 RVA: 0x0000C92C File Offset: 0x0000AB2C
		public void Clear()
		{
		}

		// Token: 0x060002F3 RID: 755 RVA: 0x0000C92E File Offset: 0x0000AB2E
		private SpecificationInfo GetSpecificationInfo()
		{
			return SettingManager.Get().GetTzpManger(this._key).SpecificationInfo;
		}

		// Token: 0x04000113 RID: 275
		private Dictionary<FormSpecModel, int> _list;

		// Token: 0x04000114 RID: 276
		private XmlElement _container;

		// Token: 0x04000115 RID: 277
		private PackageKey _key;

		// Token: 0x04000116 RID: 278
		private bool isCut;
	}
}
