using System;
using System.Collections.Generic;
using System.Linq;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.UndoRedoCommands;
using SpecDesignerCommon.ViewModel;
using UndoRedoFramework.Commands;

namespace SpecDesignerCommon.UndoRedo
{
	// Token: 0x02000007 RID: 7
	public class AddToContainerUndoRedoCommand : AbstractComplexTriggerUndoRedoCommand
	{
		// Token: 0x06000042 RID: 66 RVA: 0x000031F4 File Offset: 0x000013F4
		public AddToContainerUndoRedoCommand(List<XmlElement> elements, XmlElement container, ComponentType type)
		{
			this._container = container;
			if (this._container == null)
			{
				return;
			}
			this._newContainer = ComponentFactory.CreateEmptyComponent(this._container.Key, type, "");
			this._elements = new Dictionary<XmlElement, AddToContainerUndoRedoCommand.ComponentDimension>();
			foreach (XmlElement xmlElement in elements)
			{
				int num = -1;
				ComponentType type2 = container.Type;
				if (type2 == ComponentType.HBox || type2 == ComponentType.VBox)
				{
					num = xmlElement.Index;
				}
				this._elements.Add(xmlElement, new AddToContainerUndoRedoCommand.ComponentDimension(xmlElement.GridX, xmlElement.GridY, xmlElement.GridWidth, xmlElement.GridHeight, num));
			}
		}

		// Token: 0x06000043 RID: 67 RVA: 0x000032BC File Offset: 0x000014BC
		public override void Undo()
		{
			this._container.RemoveNode(this._newContainer);
			SettingManager.Get().GetTzpManger(this._container.Key).SpecificationInfo.Remove(this._newContainer.Name);
			foreach (KeyValuePair<XmlElement, AddToContainerUndoRedoCommand.ComponentDimension> keyValuePair in this._elements)
			{
				XmlElement key = keyValuePair.Key;
				key.GridX = (key.GridY = 0);
				if (keyValuePair.Value.GridIndex != -1)
				{
					this._container.AddNodeAt(key, keyValuePair.Value.GridIndex);
				}
				else
				{
					this._container.AddNode(key);
				}
				key.GridX = keyValuePair.Value.GridX;
				key.GridY = keyValuePair.Value.GridY;
				key.GridWidth = keyValuePair.Value.GridWidth;
				key.GridHeight = keyValuePair.Value.GridHeight;
			}
			ComponentHelper.Get(this._container.Key).MultipleSelection(this._elements.Keys);
		}

		// Token: 0x06000044 RID: 68 RVA: 0x00003400 File Offset: 0x00001600
		public override void Execute()
		{
			FormSizeComplexUndoRedoCommand formSizeComplexUndoRedoCommand = new FormSizeComplexUndoRedoCommand(this);
			SettingManager.Get().GetUndoRedoManager(this._container.Key).StartGroup(formSizeComplexUndoRedoCommand);
			int num = int.MaxValue;
			int num2 = int.MaxValue;
			ComponentType type = this._container.Type;
			if (type == ComponentType.HBox || type == ComponentType.VBox)
			{
				num = this._elements.Keys.FirstOrDefault<XmlElement>().GridX;
				num2 = this._elements.Keys.FirstOrDefault<XmlElement>().GridY;
			}
			else
			{
				foreach (XmlElement xmlElement in this._elements.Keys)
				{
					num = Math.Min(num, xmlElement.GridX);
					num2 = Math.Min(num2, xmlElement.GridY);
				}
			}
			foreach (XmlElement xmlElement2 in this._elements.Keys)
			{
				xmlElement2.Parent.RemoveNode(xmlElement2, false);
				xmlElement2.GridX -= ((int.MaxValue == num) ? xmlElement2.GridX : num);
				xmlElement2.GridY -= ((int.MaxValue == num2) ? xmlElement2.GridY : num2);
			}
			foreach (XmlElement xmlElement3 in this._elements.Keys)
			{
				ComponentType type2 = this._newContainer.Type;
				if (type2 == ComponentType.HBox || type2 == ComponentType.VBox)
				{
					this._newContainer.AddNodeAt(xmlElement3, this._newContainer.Nodes.Count);
				}
				else
				{
					this._newContainer.AddNode(xmlElement3);
				}
			}
			ComponentType type3 = this._container.Type;
			if (type3 == ComponentType.HBox || type3 == ComponentType.VBox)
			{
				this._container.AddNodeAt(this._newContainer, this._elements.FirstOrDefault<KeyValuePair<XmlElement, AddToContainerUndoRedoCommand.ComponentDimension>>().Value.GridIndex);
			}
			else
			{
				this._newContainer.GridX = ((int.MaxValue == num) ? 0 : num);
				this._newContainer.GridY = ((int.MaxValue == num2) ? 0 : num2);
				this._container.AddNode(this._newContainer);
			}
			SettingManager.Get().GetTzpManger(this._container.Key).SpecificationInfo.Add(this._newContainer);
			SettingManager.Get().GetUndoRedoManager(this._container.Key).EndGroup(formSizeComplexUndoRedoCommand);
			ComponentHelper.Get(this._container.Key).AddSelection(this._newContainer, false);
		}

		// Token: 0x06000045 RID: 69 RVA: 0x000036D8 File Offset: 0x000018D8
		public override void Clear()
		{
			this._elements.Clear();
		}

		// Token: 0x04000019 RID: 25
		private XmlElement _container;

		// Token: 0x0400001A RID: 26
		private XmlElement _newContainer;

		// Token: 0x0400001B RID: 27
		private Dictionary<XmlElement, AddToContainerUndoRedoCommand.ComponentDimension> _elements;

		// Token: 0x02000008 RID: 8
		private class ComponentDimension
		{
			// Token: 0x1700000A RID: 10
			// (get) Token: 0x06000046 RID: 70 RVA: 0x000036E5 File Offset: 0x000018E5
			// (set) Token: 0x06000047 RID: 71 RVA: 0x000036ED File Offset: 0x000018ED
			public int GridX { get; private set; }

			// Token: 0x1700000B RID: 11
			// (get) Token: 0x06000048 RID: 72 RVA: 0x000036F6 File Offset: 0x000018F6
			// (set) Token: 0x06000049 RID: 73 RVA: 0x000036FE File Offset: 0x000018FE
			public int GridY { get; private set; }

			// Token: 0x1700000C RID: 12
			// (get) Token: 0x0600004A RID: 74 RVA: 0x00003707 File Offset: 0x00001907
			// (set) Token: 0x0600004B RID: 75 RVA: 0x0000370F File Offset: 0x0000190F
			public int GridWidth { get; private set; }

			// Token: 0x1700000D RID: 13
			// (get) Token: 0x0600004C RID: 76 RVA: 0x00003718 File Offset: 0x00001918
			// (set) Token: 0x0600004D RID: 77 RVA: 0x00003720 File Offset: 0x00001920
			public int GridHeight { get; private set; }

			// Token: 0x1700000E RID: 14
			// (get) Token: 0x0600004E RID: 78 RVA: 0x00003729 File Offset: 0x00001929
			// (set) Token: 0x0600004F RID: 79 RVA: 0x00003731 File Offset: 0x00001931
			public int GridIndex { get; private set; }

			// Token: 0x06000050 RID: 80 RVA: 0x0000373A File Offset: 0x0000193A
			public ComponentDimension(int x, int y, int width, int height, int index)
			{
				this.GridX = x;
				this.GridY = y;
				this.GridWidth = width;
				this.GridHeight = height;
				this.GridIndex = index;
			}
		}
	}
}
