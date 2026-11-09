using System;
using System.Collections.Generic;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;
using UndoRedoFramework;
using UndoRedoFramework.Commands;

namespace SpecDesignerCommon.UndoRedo
{
	// Token: 0x0200007E RID: 126
	public class SpecAttributeUndoRedoCommand : AbstractComplexTriggerUndoRedoCommand
	{
		// Token: 0x060004E0 RID: 1248 RVA: 0x00015EC0 File Offset: 0x000140C0
		public SpecAttributeUndoRedoCommand(AbstractSpecNode node)
		{
			this._specNode = node;
			this._key = node.ProgramKey;
			FormSpecModel formSpecModel = this.GetSpecificationInfo().FindNodeByName(this._specNode.Name);
			this._form = ((formSpecModel == null) ? null : formSpecModel.GeneroComponent);
		}

		// Token: 0x060004E1 RID: 1249 RVA: 0x00015F28 File Offset: 0x00014128
		public void AddAttributeChanged(string attr, string oldValue, string newValue)
		{
			if (oldValue == newValue)
			{
				return;
			}
			if (!this._oldAttributes.ContainsKey(attr))
			{
				this._oldAttributes.Add(attr, oldValue);
			}
			if (!this._newAttributes.ContainsKey(attr))
			{
				this._newAttributes.Add(attr, newValue);
			}
		}

		// Token: 0x060004E2 RID: 1250 RVA: 0x00015F78 File Offset: 0x00014178
		public override void Undo()
		{
			foreach (KeyValuePair<string, string> keyValuePair in this._oldAttributes)
			{
				this._specNode.SetAttribute(keyValuePair.Key, keyValuePair.Value);
				string key;
				switch (key = keyValuePair.Key)
				{
				case "table":
				case "column":
					SpecNodeTransform.TransformTableColumn(this._specNode, this._form);
					break;
				case "lang_table":
				case "lang_rtn":
				{
					SpecMultiLangNode specMultiLangNode = this._specNode as SpecMultiLangNode;
					if (specMultiLangNode != null)
					{
						string langTable = specMultiLangNode.LangTable;
						string langRTN = specMultiLangNode.LangRTN;
						if (this._form.GetAttribute("sqlTabName") != langTable)
						{
							this._form.SetAttribute("sqlTabName", langTable);
						}
						if (this._form.GetAttribute("colName") != langRTN)
						{
							this._form.SetAttribute("colName", langRTN);
						}
						SpecNodeTransform.TransformFieldType(this._form);
					}
					break;
				}
				case "ref_table":
				case "ref_rtn":
				{
					SpecReferenceNode specReferenceNode = this._specNode as SpecReferenceNode;
					if (specReferenceNode != null)
					{
						string refTable = specReferenceNode.RefTable;
						string refRtn = specReferenceNode.RefRtn;
						if (this._form.GetAttribute("sqlTabName") != refTable)
						{
							this._form.SetAttribute("sqlTabName", refTable);
						}
						if (this._form.GetAttribute("colName") != refRtn)
						{
							this._form.SetAttribute("colName", refRtn);
						}
						SpecNodeTransform.TransformFieldType(this._form);
					}
					break;
				}
				case "req":
					SpecNodeTransform.TransformRequired(this._specNode, this._form);
					break;
				case "name":
					this.GetSpecificationInfo().Rename(this._newAttributes[keyValuePair.Key], keyValuePair.Value);
					break;
				case "can_edit":
					SpecNodeTransform.TransformCanEdit(this._specNode, this._form);
					break;
				}
			}
			if (this._specNode is SpecProgRelNode)
			{
				SpecNodeTransform.TransformProgRelNode(this._specNode as SpecProgRelNode, this._form);
				this._form.OnPropertyChanged("");
			}
			if (this._specNode is SpecReferenceNode)
			{
				SpecNodeTransform.TransformReferenceNode(this._specNode as SpecReferenceNode, this._form);
				this._form.OnPropertyChanged("");
			}
			this._specNode.OnPropertyChanged("");
			ComponentHelper.Get(this._key).AddSelection(this._form, false);
		}

		// Token: 0x060004E3 RID: 1251 RVA: 0x000162C0 File Offset: 0x000144C0
		public override void Execute()
		{
			GeneralComplexCommand generalComplexCommand = new GeneralComplexCommand(this);
			this.GetUndoRedoManager().StartGroup(generalComplexCommand);
			foreach (KeyValuePair<string, string> keyValuePair in this._newAttributes)
			{
				this._specNode.SetAttribute(keyValuePair.Key, keyValuePair.Value);
				string key;
				switch (key = keyValuePair.Key)
				{
				case "column":
					SpecNodeTransform.TransformTableColumn(this._specNode, this._form);
					break;
				case "lang_table":
				case "lang_rtn":
				{
					SpecMultiLangNode specMultiLangNode = this._specNode as SpecMultiLangNode;
					if (specMultiLangNode != null)
					{
						string langTable = specMultiLangNode.LangTable;
						string langRTN = specMultiLangNode.LangRTN;
						if (this._form.GetAttribute("sqlTabName") != langTable)
						{
							this._form.SetAttribute("sqlTabName", langTable);
						}
						if (this._form.GetAttribute("colName") != langRTN)
						{
							this._form.SetAttribute("colName", langRTN);
						}
						SpecNodeTransform.TransformFieldType(this._form);
					}
					break;
				}
				case "ref_table":
				case "ref_rtn":
				{
					SpecReferenceNode specReferenceNode = this._specNode as SpecReferenceNode;
					if (specReferenceNode != null)
					{
						string refTable = specReferenceNode.RefTable;
						string refRtn = specReferenceNode.RefRtn;
						if (this._form.GetAttribute("sqlTabName") != refTable)
						{
							this._form.SetAttribute("sqlTabName", refTable);
						}
						if (this._form.GetAttribute("colName") != refRtn)
						{
							this._form.SetAttribute("colName", refRtn);
						}
						SpecNodeTransform.TransformFieldType(this._form);
					}
					break;
				}
				case "req":
					SpecNodeTransform.TransformRequired(this._specNode, this._form);
					break;
				case "name":
					this.GetSpecificationInfo().Rename(this._oldAttributes[keyValuePair.Key], keyValuePair.Value);
					break;
				case "can_edit":
					SpecNodeTransform.TransformCanEdit(this._specNode, this._form);
					break;
				}
			}
			if (this._specNode is SpecProgRelNode)
			{
				SpecNodeTransform.TransformProgRelNode(this._specNode as SpecProgRelNode, this._form);
				this._form.OnPropertyChanged("");
			}
			if (this._specNode is SpecReferenceNode)
			{
				SpecNodeTransform.TransformReferenceNode(this._specNode as SpecReferenceNode, this._form);
				this._form.OnPropertyChanged("");
			}
			if (this._specNode is SpecHelpCodeNode)
			{
				FormSpecModel formSpecModel = this.GetSpecificationInfo().FindNodeByName(this._specNode.Name);
				if (formSpecModel.SpecField != null)
				{
					formSpecModel.SpecField.OnPropertyChanged("Status");
				}
			}
			this._specNode.OnPropertyChanged("");
			if (this._form != null)
			{
				this._form.OnPropertyChanged("SpecNodeStatus");
			}
			this.GetUndoRedoManager().EndGroup(generalComplexCommand);
			ComponentHelper.Get(this._key).AddSelection(this._form, false);
		}

		// Token: 0x060004E4 RID: 1252 RVA: 0x00016670 File Offset: 0x00014870
		private SpecificationInfo GetSpecificationInfo()
		{
			return SettingManager.Get().GetTzpManger(this._key).SpecificationInfo;
		}

		// Token: 0x060004E5 RID: 1253 RVA: 0x00016687 File Offset: 0x00014887
		private UndoRedoManager GetUndoRedoManager()
		{
			return SettingManager.Get().GetUndoRedoManager(this._key);
		}

		// Token: 0x060004E6 RID: 1254 RVA: 0x00016699 File Offset: 0x00014899
		public override void Clear()
		{
			if (this._newAttributes != null)
			{
				this._newAttributes.Clear();
			}
			if (this._oldAttributes != null)
			{
				this._oldAttributes.Clear();
			}
		}

		// Token: 0x040001D1 RID: 465
		private AbstractSpecNode _specNode;

		// Token: 0x040001D2 RID: 466
		private XmlElement _form;

		// Token: 0x040001D3 RID: 467
		private Dictionary<string, string> _oldAttributes = new Dictionary<string, string>();

		// Token: 0x040001D4 RID: 468
		private Dictionary<string, string> _newAttributes = new Dictionary<string, string>();

		// Token: 0x040001D5 RID: 469
		private PackageKey _key;
	}
}
