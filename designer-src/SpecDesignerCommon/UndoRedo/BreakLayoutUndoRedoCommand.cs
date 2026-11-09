using System;
using System.Collections.Generic;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.UndoRedoCommands;
using SpecDesignerCommon.ViewModel;
using UndoRedoFramework.Commands;

namespace SpecDesignerCommon.UndoRedo
{
	// Token: 0x020000E7 RID: 231
	public class BreakLayoutUndoRedoCommand : AbstractComplexTriggerUndoRedoCommand
	{
		// Token: 0x060007C0 RID: 1984 RVA: 0x000227E8 File Offset: 0x000209E8
		public BreakLayoutUndoRedoCommand(XmlElement box)
		{
			this._box = box;
			this._container = this._box.Parent;
			this._key = this._box.Key;
			this._insertIndex = this._box.Index;
			if (ComponentType.HBox != this._container.Type && ComponentType.VBox != this._container.Type)
			{
				this._oriX = this._box.GridX;
				this._oriY = this._box.GridY;
			}
			ComponentType type = this._box.Type;
			if (type != ComponentType.HBox)
			{
				if (type != ComponentType.VBox)
				{
					return;
				}
				this._oriSize = this._box.GridWidth;
			}
			else
			{
				this._oriSize = this._box.GridHeight;
			}
			this._children = new List<XmlElement>();
			foreach (XmlElement xmlElement in this._box.Nodes)
			{
				this._children.Add(xmlElement);
			}
		}

		// Token: 0x060007C1 RID: 1985 RVA: 0x00022920 File Offset: 0x00020B20
		public override void Undo()
		{
			FormSizeComplexUndoRedoCommand formSizeComplexUndoRedoCommand = new FormSizeComplexUndoRedoCommand(this);
			SettingManager.Get().GetUndoRedoManager(this._key).StartGroup(formSizeComplexUndoRedoCommand);
			int num = int.MaxValue;
			int num2 = int.MaxValue;
			XmlElement xmlElement = this._children[0];
			ComponentType type = this._box.Type;
			if (type != ComponentType.HBox)
			{
				if (type == ComponentType.VBox)
				{
					foreach (XmlElement xmlElement2 in this._children)
					{
						int gridX = xmlElement2.GridX;
						int gridY = xmlElement2.GridY;
						int gridHeight = xmlElement2.GridHeight;
						int gridWidth = xmlElement2.GridWidth;
						num = Math.Min(num, gridX);
						num2 = Math.Min(num2, gridY);
						this._container.RemoveNode(xmlElement2);
					}
					if (this._container.NodeName == ComponentType.HBox.ToString() || this._container.NodeName == ComponentType.VBox.ToString())
					{
						this._container.AddNodeAt(this._box, this._insertIndex);
					}
					else
					{
						this._container.AddNode(this._box);
					}
					this._box.GridX = num;
					this._box.GridY = num2;
					for (int i = 0; i < this._children.Count; i++)
					{
						XmlElement xmlElement3 = this._children[i];
						xmlElement3.GridWidth = this._oriSize;
						this._box.AddNodeAt(xmlElement3, i);
					}
					this._box.GridWidth = this._oriSize;
					this._box.GridWidth = this._children[0].GridWidth;
				}
			}
			else
			{
				foreach (XmlElement xmlElement4 in this._children)
				{
					num = Math.Min(num, xmlElement4.GridX);
					num2 = Math.Min(num2, xmlElement4.GridY);
					this._container.RemoveNode(xmlElement4);
				}
				if (this._container.NodeName == ComponentType.HBox.ToString() || this._container.NodeName == ComponentType.VBox.ToString())
				{
					this._container.AddNodeAt(this._box, this._insertIndex);
				}
				else
				{
					this._container.AddNode(this._box);
				}
				this._box.GridX = num;
				this._box.GridY = num2;
				for (int j = 0; j < this._children.Count; j++)
				{
					XmlElement xmlElement5 = this._children[j];
					xmlElement5.GridHeight = this._oriSize;
					this._box.AddNodeAt(xmlElement5, j);
				}
				this._box.GridHeight = this._oriSize;
				this._box.GridHeight = this._children[0].GridHeight;
			}
			this.GetSpecificationInfo().Add(this._box);
			SettingManager.Get().GetUndoRedoManager(this._key).EndGroup(formSizeComplexUndoRedoCommand);
			ComponentHelper.Get(this._box.Key).AddSelection(this._box, false);
		}

		// Token: 0x060007C2 RID: 1986 RVA: 0x00022C8C File Offset: 0x00020E8C
		public override void Execute()
		{
			FormSizeComplexUndoRedoCommand formSizeComplexUndoRedoCommand = new FormSizeComplexUndoRedoCommand(this, false);
			SettingManager.Get().GetUndoRedoManager(this._key).StartGroup(formSizeComplexUndoRedoCommand);
			this._container.RemoveNode(this._box);
			this.GetSpecificationInfo().Remove(this._box.Name);
			int num = 0;
			ComponentType type = this._box.Type;
			foreach (XmlElement xmlElement in this._children)
			{
				this._container.AddNode(xmlElement);
				ComponentType componentType = type;
				if (componentType != ComponentType.HBox)
				{
					if (componentType == ComponentType.VBox)
					{
						xmlElement.GridX = this._box.GridX;
						xmlElement.GridY = num + this._oriY;
						xmlElement.GridWidth = this._oriSize;
						num += xmlElement.GridHeight;
					}
				}
				else
				{
					xmlElement.GridY = this._box.GridY;
					xmlElement.GridX = num + this._oriX;
					xmlElement.GridHeight = this._oriSize;
					num += xmlElement.GridWidth;
				}
			}
			SettingManager.Get().GetUndoRedoManager(this._key).EndGroup(formSizeComplexUndoRedoCommand);
			ComponentHelper.Get(this._box.Key).AddSelection(this._container, false);
		}

		// Token: 0x060007C3 RID: 1987 RVA: 0x00022DE8 File Offset: 0x00020FE8
		private SpecificationInfo GetSpecificationInfo()
		{
			return SettingManager.Get().GetTzpManger(this._box.Key).SpecificationInfo;
		}

		// Token: 0x060007C4 RID: 1988 RVA: 0x00022E04 File Offset: 0x00021004
		public override void Clear()
		{
			if (this._children != null)
			{
				this._children.Clear();
			}
		}

		// Token: 0x040002BA RID: 698
		private XmlElement _box;

		// Token: 0x040002BB RID: 699
		private XmlElement _container;

		// Token: 0x040002BC RID: 700
		private int _oriSize = -1;

		// Token: 0x040002BD RID: 701
		private int _oriX = -1;

		// Token: 0x040002BE RID: 702
		private int _oriY = -1;

		// Token: 0x040002BF RID: 703
		private List<XmlElement> _children;

		// Token: 0x040002C0 RID: 704
		private int _insertIndex = -1;

		// Token: 0x040002C1 RID: 705
		private PackageKey _key;
	}
}
