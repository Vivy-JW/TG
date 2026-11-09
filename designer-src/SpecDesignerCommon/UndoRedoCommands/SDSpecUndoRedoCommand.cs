using System;
using System.Xml.Linq;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;
using UndoRedoFramework.Commands;

namespace SpecDesignerCommon.UndoRedoCommands
{
	// Token: 0x0200004A RID: 74
	public class SDSpecUndoRedoCommand : IUndoRedoCommand
	{
		// Token: 0x0600027F RID: 639 RVA: 0x0000B3B4 File Offset: 0x000095B4
		public SDSpecUndoRedoCommand(AbstractSpecNode specNode, string newContent)
		{
			this._specNode = specNode;
			if (this._specNode != null)
			{
				this._key = this._specNode.ProgramKey;
				this._oldContent = this._specNode.CDATA;
				this._newContent = newContent;
				if (this._specNode.Name != null)
				{
					FormSpecModel formSpecModel = SettingManager.Get().GetTzpManger(this._key).SpecificationInfo.FindNodeByName(this._specNode.Name);
					if (formSpecModel != null)
					{
						this._form = formSpecModel.GeneroComponent;
					}
				}
			}
		}

		// Token: 0x06000280 RID: 640 RVA: 0x0000B457 File Offset: 0x00009657
		public void Undo()
		{
			this.ChangeContent(this._oldContent);
		}

		// Token: 0x06000281 RID: 641 RVA: 0x0000B465 File Offset: 0x00009665
		public void Execute()
		{
			this.ChangeContent(this._newContent);
		}

		// Token: 0x06000282 RID: 642 RVA: 0x0000B474 File Offset: 0x00009674
		private void ChangeContent(string content)
		{
			this._specNode.Source.ReplaceNodes(new XCData(content));
			this._specNode.OnPropertyChanged("CDATA");
			if (this._form != null)
			{
				this._form.OnPropertyChanged("IsSpecificationDefined");
				ComponentHelper.Get(this._key).AddSelection(this._form, false);
			}
		}

		// Token: 0x06000283 RID: 643 RVA: 0x0000B4D6 File Offset: 0x000096D6
		public void Clear()
		{
			this._newContent = string.Empty;
			this._oldContent = string.Empty;
		}

		// Token: 0x040000F0 RID: 240
		private AbstractSpecNode _specNode;

		// Token: 0x040000F1 RID: 241
		private XmlElement _form;

		// Token: 0x040000F2 RID: 242
		private PackageKey _key;

		// Token: 0x040000F3 RID: 243
		private string _newContent = string.Empty;

		// Token: 0x040000F4 RID: 244
		private string _oldContent = string.Empty;
	}
}
