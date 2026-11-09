using System;
using System.Collections.Generic;
using System.Xml.Linq;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.UndoRedoCommands;
using SpecDesignerCommon.ViewModel;
using UndoRedoFramework.Commands;

namespace SpecDesignerCommon.UndoRedo
{
	// Token: 0x0200007F RID: 127
	public class AddComponetsUndoRedoCommand : AbstractComplexTriggerUndoRedoCommand
	{
		// Token: 0x060004E7 RID: 1255 RVA: 0x000166C1 File Offset: 0x000148C1
		public AddComponetsUndoRedoCommand(IEnumerable<XmlElement> list, XmlElement container)
			: this(list, container, 0)
		{
		}

		// Token: 0x060004E8 RID: 1256 RVA: 0x000166CC File Offset: 0x000148CC
		public AddComponetsUndoRedoCommand(IEnumerable<XmlElement> list, XmlElement container, int index)
		{
			this._children = new List<XmlElement>();
			foreach (XmlElement xmlElement in list)
			{
				this._children.Add(xmlElement);
			}
			this._container = container;
			this._index = index;
		}

		// Token: 0x060004E9 RID: 1257 RVA: 0x00016738 File Offset: 0x00014938
		public AddComponetsUndoRedoCommand(IEnumerable<XmlElement> list, XmlElement container, int posX, int posY)
		{
			this._list = new Dictionary<string, AddComponetsUndoRedoCommand.ComponentDimension>();
			this._children = new List<XmlElement>();
			foreach (XmlElement xmlElement in list)
			{
				this._children.Add(xmlElement);
				this._list.Add(xmlElement.Name, new AddComponetsUndoRedoCommand.ComponentDimension(xmlElement.GridX, xmlElement.GridY));
			}
			this._container = container;
			this._index = 0;
			this._posX = posX;
			this._posY = posY;
		}

		// Token: 0x060004EA RID: 1258 RVA: 0x000167E0 File Offset: 0x000149E0
		public override void Undo()
		{
			FormSizeComplexUndoRedoCommand formSizeComplexUndoRedoCommand = new FormSizeComplexUndoRedoCommand(this);
			SettingManager.Get().GetUndoRedoManager(this._container.Key).StartGroup(formSizeComplexUndoRedoCommand);
			foreach (XmlElement xmlElement in this._children)
			{
				this.RemoveSpecContainsChildren(xmlElement);
				this._container.RemoveNode(xmlElement);
				if (this._list != null && this._list.ContainsKey(xmlElement.Name))
				{
					xmlElement.GridX = this._list[xmlElement.Name].X;
					xmlElement.GridY = this._list[xmlElement.Name].Y;
				}
			}
			SettingManager.Get().GetUndoRedoManager(this._container.Key).EndGroup(formSizeComplexUndoRedoCommand);
			ComponentHelper.Get(this._container.Key).AddSelection(this._container, false);
		}

		// Token: 0x060004EB RID: 1259 RVA: 0x000168EC File Offset: 0x00014AEC
		public override void Execute()
		{
			FormSizeComplexUndoRedoCommand formSizeComplexUndoRedoCommand = new FormSizeComplexUndoRedoCommand(this);
			SettingManager.Get().GetUndoRedoManager(this._container.Key).StartGroup(formSizeComplexUndoRedoCommand);
			int num = 0;
			int num2 = 0;
			int i = 0;
			while (i < this._children.Count)
			{
				XmlElement xmlElement = this._children[i];
				ComponentType type = this._container.Type;
				if (type == ComponentType.Folder)
				{
					goto IL_0074;
				}
				switch (type)
				{
				case ComponentType.HBox:
				case ComponentType.Table:
				case ComponentType.Tree:
				case ComponentType.VBox:
					goto IL_0074;
				}
				if (i == 0)
				{
					num = xmlElement.GridX;
					num2 = xmlElement.GridY;
				}
				if (i == 0)
				{
					this._container.AddNode(xmlElement);
					num = xmlElement.GridX - num;
					num2 = xmlElement.GridY - num2;
					xmlElement.GridX += this._posX;
					xmlElement.GridY += this._posY;
				}
				else
				{
					xmlElement.GridX += this._posX + num;
					xmlElement.GridY += this._posY + num2;
					this._container.AddNode(xmlElement);
				}
				IL_0126:
				this.AddChildrenForSpec(xmlElement);
				i++;
				continue;
				IL_0074:
				this._container.AddNodeAt(xmlElement, this._index);
				goto IL_0126;
			}
			for (int j = 0; j < this._children.Count; j++)
			{
				XmlElement xmlElement2 = this._children[j];
				FormSpecModel formSpecModel = this.GetSpecificationInfo().FindNodeByName(xmlElement2.Name);
				if (formSpecModel.SpecField != null)
				{
					XElement columnInfo = TableColumnHelper.GetColumnInfo(formSpecModel.SpecField.Table, formSpecModel.SpecField.Column);
					XElement columnAttrInfo = TableColumnHelper.GetColumnAttrInfo(formSpecModel.SpecField.Table, formSpecModel.SpecField.Column);
					if (columnInfo != null)
					{
						formSpecModel.SpecField.SetAttribute("attribute", columnInfo.Attribute("attribute").Value);
						formSpecModel.SpecField.SetAttribute("type", columnInfo.Attribute("type").Value);
						formSpecModel.SpecField.SetAttribute("req", columnInfo.Attribute("req").Value);
						if (columnAttrInfo != null)
						{
							formSpecModel.SpecField.SetAttribute("i_zoom", columnAttrInfo.Attribute("i_zoom").Value);
							formSpecModel.SpecField.SetAttribute("c_zoom", columnAttrInfo.Attribute("c_zoom").Value);
							formSpecModel.SpecField.SetAttribute("default", columnAttrInfo.Attribute("default").Value);
							formSpecModel.SpecField.SetAttribute("max", columnAttrInfo.Attribute("max").Value);
							formSpecModel.SpecField.SetAttribute("min", columnAttrInfo.Attribute("min").Value);
							formSpecModel.SpecField.SetAttribute("chk_ref", columnAttrInfo.Attribute("chk_ref").Value);
							formSpecModel.SpecField.SetAttribute("items", columnAttrInfo.Attribute("items").Value);
						}
					}
				}
			}
			SettingManager.Get().GetUndoRedoManager(this._container.Key).EndGroup(formSizeComplexUndoRedoCommand);
			ComponentHelper.Get(this._container.Key).MultipleSelection(this._children);
		}

		// Token: 0x060004EC RID: 1260 RVA: 0x00016C94 File Offset: 0x00014E94
		private void RemoveSpecContainsChildren(XmlElement children)
		{
			for (int i = 0; i < children.Nodes.Count; i++)
			{
				XmlElement xmlElement = children.Nodes[i];
				this.RemoveSpecContainsChildren(xmlElement);
			}
			this.GetSpecificationInfo().Remove(children.Name);
		}

		// Token: 0x060004ED RID: 1261 RVA: 0x00016CDC File Offset: 0x00014EDC
		private void AddChildrenForSpec(XmlElement children)
		{
			this.GetSpecificationInfo().Add(children, true);
			foreach (XmlElement xmlElement in children.Nodes)
			{
				this.AddChildrenForSpec(xmlElement);
			}
		}

		// Token: 0x060004EE RID: 1262 RVA: 0x00016D38 File Offset: 0x00014F38
		public override void Clear()
		{
			if (this._children != null)
			{
				this._children.Clear();
			}
			if (this._list != null)
			{
				this._list.Clear();
			}
		}

		// Token: 0x060004EF RID: 1263 RVA: 0x00016D60 File Offset: 0x00014F60
		private SpecificationInfo GetSpecificationInfo()
		{
			if (this._container != null)
			{
				return SettingManager.Get().GetTzpManger(this._container.Key).SpecificationInfo;
			}
			return null;
		}

		// Token: 0x040001D6 RID: 470
		private Dictionary<string, AddComponetsUndoRedoCommand.ComponentDimension> _list;

		// Token: 0x040001D7 RID: 471
		private List<XmlElement> _children;

		// Token: 0x040001D8 RID: 472
		private XmlElement _container;

		// Token: 0x040001D9 RID: 473
		private int _index;

		// Token: 0x040001DA RID: 474
		private int _posX;

		// Token: 0x040001DB RID: 475
		private int _posY;

		// Token: 0x02000080 RID: 128
		private class ComponentDimension
		{
			// Token: 0x1700015D RID: 349
			// (get) Token: 0x060004F0 RID: 1264 RVA: 0x00016D86 File Offset: 0x00014F86
			// (set) Token: 0x060004F1 RID: 1265 RVA: 0x00016D8E File Offset: 0x00014F8E
			public int X { get; private set; }

			// Token: 0x1700015E RID: 350
			// (get) Token: 0x060004F2 RID: 1266 RVA: 0x00016D97 File Offset: 0x00014F97
			// (set) Token: 0x060004F3 RID: 1267 RVA: 0x00016D9F File Offset: 0x00014F9F
			public int Y { get; private set; }

			// Token: 0x060004F4 RID: 1268 RVA: 0x00016DA8 File Offset: 0x00014FA8
			public ComponentDimension(int x, int y)
			{
				this.X = x;
				this.Y = y;
			}
		}
	}
}
