using System;
using System.Collections.Generic;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;
using UndoRedoFramework;
using UndoRedoFramework.Commands;

namespace SpecDesignerCommon.UndoRedo
{
	// Token: 0x02000097 RID: 151
	public class ConvertContainerTypeUndoRedoCommand : AbstractComplexTriggerUndoRedoCommand
	{
		// Token: 0x0600062E RID: 1582 RVA: 0x0001C000 File Offset: 0x0001A200
		public ConvertContainerTypeUndoRedoCommand(XmlElement element, ComponentType type)
		{
			if (element == null)
			{
				return;
			}
			this._oldElement = element;
			this._parentElement = element.Parent;
			this._index = this._oldElement.Index;
			this._newElement = ComponentFactory.ConvertTo(this._oldElement, type);
			this._height = element.GridHeight;
			this._key = element.Key;
			this._children = new List<ConvertContainerTypeUndoRedoCommand.ComponentDimension>();
			foreach (XmlElement xmlElement in element.Nodes)
			{
				this._children.Add(new ConvertContainerTypeUndoRedoCommand.ComponentDimension(xmlElement, xmlElement.GridY));
			}
		}

		// Token: 0x0600062F RID: 1583 RVA: 0x0001C0C0 File Offset: 0x0001A2C0
		public override void Undo()
		{
			ComponentHelper.Get(this._key).ClearSelection();
			GeneralComplexCommand generalComplexCommand = new GeneralComplexCommand(this);
			this.GetUndoRedoManager().StartGroup(generalComplexCommand);
			this.GetSpecificationInfo().ConvertComponentType(this._newElement, this._oldElement);
			foreach (ConvertContainerTypeUndoRedoCommand.ComponentDimension componentDimension in this._children)
			{
				this._oldElement.AddNode(componentDimension.Element);
				componentDimension.Element.GridY = componentDimension.Y;
			}
			this._oldElement.GridHeight = this._height;
			if (this._parentElement != null)
			{
				ComponentType type = this._parentElement.Type;
				if (type == ComponentType.HBox || type == ComponentType.VBox)
				{
					this._parentElement.AddNodeAt(this._oldElement, this._index);
				}
				else
				{
					this._parentElement.AddNode(this._oldElement);
				}
			}
			this.GetUndoRedoManager().EndGroup(generalComplexCommand);
			ComponentHelper.Get(this._key).AddSelection(this._oldElement, false);
		}

		// Token: 0x06000630 RID: 1584 RVA: 0x0001C1E4 File Offset: 0x0001A3E4
		public override void Execute()
		{
			ComponentHelper.Get(this._key).ClearSelection();
			GeneralComplexCommand generalComplexCommand = new GeneralComplexCommand(this, false);
			this.GetUndoRedoManager().StartGroup(generalComplexCommand);
			this.GetSpecificationInfo().ConvertComponentType(this._oldElement, this._newElement);
			int num = 0;
			for (int i = 0; i < this._children.Count; i++)
			{
				ConvertContainerTypeUndoRedoCommand.ComponentDimension componentDimension = this._children[i];
				XmlElement element = componentDimension.Element;
				if (i == 0)
				{
					num = element.GridY;
					this._newElement.AddNode(element);
					num = element.GridY - num;
				}
				else
				{
					element.GridY += num;
					this._newElement.AddNode(element);
				}
			}
			if (this._parentElement != null)
			{
				ComponentType type = this._parentElement.Type;
				if (type == ComponentType.HBox || type == ComponentType.VBox)
				{
					this._parentElement.AddNodeAt(this._newElement, this._index);
				}
				else
				{
					this._parentElement.AddNode(this._newElement);
				}
			}
			this.GetUndoRedoManager().EndGroup(generalComplexCommand);
			ComponentHelper.Get(this._key).AddSelection(this._newElement, false);
		}

		// Token: 0x06000631 RID: 1585 RVA: 0x0001C306 File Offset: 0x0001A506
		public override void Clear()
		{
			if (this._children != null)
			{
				this._children.Clear();
			}
		}

		// Token: 0x06000632 RID: 1586 RVA: 0x0001C31B File Offset: 0x0001A51B
		private UndoRedoManager GetUndoRedoManager()
		{
			return SettingManager.Get().GetUndoRedoManager(this._key);
		}

		// Token: 0x06000633 RID: 1587 RVA: 0x0001C32D File Offset: 0x0001A52D
		private SpecificationInfo GetSpecificationInfo()
		{
			return SettingManager.Get().GetTzpManger(this._key).SpecificationInfo;
		}

		// Token: 0x0400025C RID: 604
		private XmlElement _newElement;

		// Token: 0x0400025D RID: 605
		private XmlElement _oldElement;

		// Token: 0x0400025E RID: 606
		private XmlElement _parentElement;

		// Token: 0x0400025F RID: 607
		private PackageKey _key;

		// Token: 0x04000260 RID: 608
		private int _height;

		// Token: 0x04000261 RID: 609
		private int _index;

		// Token: 0x04000262 RID: 610
		private List<ConvertContainerTypeUndoRedoCommand.ComponentDimension> _children;

		// Token: 0x02000098 RID: 152
		private class ComponentDimension
		{
			// Token: 0x170001C9 RID: 457
			// (get) Token: 0x06000634 RID: 1588 RVA: 0x0001C344 File Offset: 0x0001A544
			// (set) Token: 0x06000635 RID: 1589 RVA: 0x0001C34C File Offset: 0x0001A54C
			public int Y { get; private set; }

			// Token: 0x170001CA RID: 458
			// (get) Token: 0x06000636 RID: 1590 RVA: 0x0001C355 File Offset: 0x0001A555
			// (set) Token: 0x06000637 RID: 1591 RVA: 0x0001C35D File Offset: 0x0001A55D
			public XmlElement Element { get; private set; }

			// Token: 0x06000638 RID: 1592 RVA: 0x0001C366 File Offset: 0x0001A566
			public ComponentDimension(XmlElement element, int y)
			{
				this.Element = element;
				this.Y = y;
			}
		}
	}
}
