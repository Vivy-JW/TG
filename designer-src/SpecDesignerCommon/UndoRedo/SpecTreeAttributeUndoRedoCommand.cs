using System;
using System.Collections.Generic;
using System.Xml.Linq;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;
using UndoRedoFramework.Commands;

namespace SpecDesignerCommon.UndoRedo
{
	// Token: 0x020000FE RID: 254
	public class SpecTreeAttributeUndoRedoCommand : IUndoRedoCommand
	{
		// Token: 0x0600083F RID: 2111 RVA: 0x0002420C File Offset: 0x0002240C
		public SpecTreeAttributeUndoRedoCommand(SpecTreeNode node, string elementName, string PropertyName)
		{
			this._specNode = node;
			this._key = node.ProgramKey;
			this._propertyName = PropertyName;
			this._sourceElement = node.Source.Element(elementName);
			FormSpecModel formSpecModel = SettingManager.Get().GetTzpManger(this._key).SpecificationInfo.FindNodeByName(node.Name);
			if (formSpecModel != null)
			{
				this._form = formSpecModel.GeneroComponent;
			}
		}

		// Token: 0x06000840 RID: 2112 RVA: 0x00024298 File Offset: 0x00022498
		public void AddAttributeChanged(string attr, string newValue)
		{
			string text = ((this._sourceElement.Attribute(attr) == null) ? null : this._sourceElement.Attribute(attr).Value);
			if (text == null || text == newValue)
			{
				return;
			}
			if (!this._oldAttributes.ContainsKey(attr))
			{
				this._oldAttributes.Add(attr, text);
			}
			if (!this._newAttributes.ContainsKey(attr))
			{
				this._newAttributes.Add(attr, newValue);
			}
		}

		// Token: 0x06000841 RID: 2113 RVA: 0x00024318 File Offset: 0x00022518
		public void Undo()
		{
			foreach (KeyValuePair<string, string> keyValuePair in this._oldAttributes)
			{
				this._sourceElement.SetAttributeValue(keyValuePair.Key, keyValuePair.Value);
			}
			this._specNode.OnPropertyChanged(this._propertyName);
			if (this._form != null)
			{
				ComponentHelper.Get(this._key).AddSelection(this._form, false);
			}
		}

		// Token: 0x06000842 RID: 2114 RVA: 0x000243B4 File Offset: 0x000225B4
		public void Execute()
		{
			foreach (KeyValuePair<string, string> keyValuePair in this._newAttributes)
			{
				this._sourceElement.SetAttributeValue(keyValuePair.Key, keyValuePair.Value);
			}
			this._specNode.OnPropertyChanged(this._propertyName);
			if (this._form != null)
			{
				ComponentHelper.Get(this._key).AddSelection(this._form, false);
			}
		}

		// Token: 0x06000843 RID: 2115 RVA: 0x00024450 File Offset: 0x00022650
		private SpecificationInfo GetSpecificationInfo()
		{
			return SettingManager.Get().GetTzpManger(this._key).SpecificationInfo;
		}

		// Token: 0x06000844 RID: 2116 RVA: 0x00024467 File Offset: 0x00022667
		public void Clear()
		{
			this._newAttributes.Clear();
			this._oldAttributes.Clear();
		}

		// Token: 0x040002EA RID: 746
		private SpecTreeNode _specNode;

		// Token: 0x040002EB RID: 747
		private XElement _sourceElement;

		// Token: 0x040002EC RID: 748
		private string _propertyName;

		// Token: 0x040002ED RID: 749
		private XmlElement _form;

		// Token: 0x040002EE RID: 750
		private Dictionary<string, string> _oldAttributes = new Dictionary<string, string>();

		// Token: 0x040002EF RID: 751
		private Dictionary<string, string> _newAttributes = new Dictionary<string, string>();

		// Token: 0x040002F0 RID: 752
		private PackageKey _key;
	}
}
