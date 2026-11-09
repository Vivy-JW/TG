using System;
using System.Collections.Generic;
using System.Linq;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.UndoRedoCommands;
using SpecDesignerCommon.ViewModel;
using UndoRedoFramework.Commands;

namespace SpecDesignerCommon.UndoRedo
{
	// Token: 0x02000130 RID: 304
	public class DragComponentsUndoRedoCommand : AbstractComplexTriggerUndoRedoCommand
	{
		// Token: 0x06000AC5 RID: 2757 RVA: 0x00034E60 File Offset: 0x00033060
		public DragComponentsUndoRedoCommand(XmlElement sourceContainer)
		{
			this._oldContainer = sourceContainer;
		}

		// Token: 0x06000AC6 RID: 2758 RVA: 0x00034E90 File Offset: 0x00033090
		public void AppendSelection(XmlElement element, int index, int offsetX, int offsetY)
		{
			int num = -1;
			ComponentType type = this._oldContainer.Type;
			if (type != ComponentType.Folder)
			{
				switch (type)
				{
				case ComponentType.HBox:
				case ComponentType.Table:
				case ComponentType.Tree:
				case ComponentType.VBox:
					break;
				case ComponentType.Page:
				case ComponentType.RadioGroup:
				case ComponentType.ScrollGrid:
					goto IL_003A;
				default:
					goto IL_003A;
				}
			}
			num = index;
			IL_003A:
			DragComponentsUndoRedoCommand.ComponentDimension componentDimension = new DragComponentsUndoRedoCommand.ComponentDimension(element.GridX, element.GridY, num, offsetX, offsetY);
			this._list.Add(element, componentDimension);
		}

		// Token: 0x06000AC7 RID: 2759 RVA: 0x00034EFA File Offset: 0x000330FA
		public void AppendSelection(XmlElement element, int index)
		{
			this.AppendSelection(element, index, 0, 0);
		}

		// Token: 0x06000AC8 RID: 2760 RVA: 0x00034F08 File Offset: 0x00033108
		public void SetDestination(XmlElement destinationContainer, int posX, int posY)
		{
			this._newContainer = destinationContainer;
			this._posX = FormDesignSetting.TransformToGridWidth((double)posX);
			int num = int.MaxValue;
			foreach (XmlElement xmlElement in this._list.Keys)
			{
				if (xmlElement.GridX < num)
				{
					num = xmlElement.GridX;
				}
			}
			this._posX -= num;
			if (this._posX < 0)
			{
				this._posX = 0;
			}
			bool flag = 0 == posY % FormDesignSetting.UnitHeight;
			this._posY = FormDesignSetting.TransformToGridHeight((double)posY) + (flag ? 0 : (-1));
			ComponentType type = this._newContainer.Type;
			if (type == ComponentType.Page)
			{
				this._posY--;
			}
			this._posY -= this._list.Keys.FirstOrDefault<XmlElement>().GridY;
			if (this._posY < 0)
			{
				this._posY = 0;
			}
		}

		// Token: 0x06000AC9 RID: 2761 RVA: 0x00035014 File Offset: 0x00033214
		public void SetDestination(XmlElement destinationContainer, int index)
		{
			this._newContainer = destinationContainer;
			for (int i = 0; i < this._list.Count; i++)
			{
				this._list.ElementAtOrDefault<KeyValuePair<XmlElement, DragComponentsUndoRedoCommand.ComponentDimension>>(i).Value.NewIndex = i + index;
			}
		}

		// Token: 0x170002D4 RID: 724
		// (get) Token: 0x06000ACA RID: 2762 RVA: 0x0003505C File Offset: 0x0003325C
		public bool IsMovePages
		{
			get
			{
				foreach (XmlElement xmlElement in this._list.Keys)
				{
					ComponentType type = xmlElement.Type;
					if (type == ComponentType.Page)
					{
						return true;
					}
				}
				return false;
			}
		}

		// Token: 0x06000ACB RID: 2763 RVA: 0x000350C0 File Offset: 0x000332C0
		public override void Undo()
		{
			foreach (KeyValuePair<XmlElement, DragComponentsUndoRedoCommand.ComponentDimension> keyValuePair in this._list)
			{
				XmlElement key = keyValuePair.Key;
				if (key.Parent != null)
				{
					key.Parent.RemoveNode(key, false);
				}
			}
			foreach (KeyValuePair<XmlElement, DragComponentsUndoRedoCommand.ComponentDimension> keyValuePair2 in this._list)
			{
				XmlElement key2 = keyValuePair2.Key;
				key2.GridX = (key2.GridY = 0);
				if (-1 != keyValuePair2.Value.OldIndex)
				{
					this._oldContainer.AddNodeAt(key2, keyValuePair2.Value.OldIndex);
				}
				else
				{
					this._oldContainer.AddNode(key2);
					key2.GridX = keyValuePair2.Value.OldX;
					key2.GridY = keyValuePair2.Value.OldY;
				}
			}
			ComponentHelper.Get(this._oldContainer.Key).MultipleSelection(this._list.Keys);
		}

		// Token: 0x06000ACC RID: 2764 RVA: 0x000351FC File Offset: 0x000333FC
		public override void Execute()
		{
			FormSizeComplexUndoRedoCommand formSizeComplexUndoRedoCommand = new FormSizeComplexUndoRedoCommand(this, false);
			SettingManager.Get().GetUndoRedoManager(this._newContainer.Key).StartGroup(formSizeComplexUndoRedoCommand);
			foreach (KeyValuePair<XmlElement, DragComponentsUndoRedoCommand.ComponentDimension> keyValuePair in this._list)
			{
				XmlElement key = keyValuePair.Key;
				this._newContainer.AddNodeAt(key, (-1 == keyValuePair.Value.NewIndex) ? 0 : keyValuePair.Value.NewIndex);
				if (-2147483648 != this._posX && -2147483648 != this._posY)
				{
					key.GridX = keyValuePair.Value.OldX - keyValuePair.Value.OffsetX + this._posX;
					key.GridY = keyValuePair.Value.OldY - keyValuePair.Value.OffsetY + this._posY;
				}
			}
			SettingManager.Get().GetUndoRedoManager(this._newContainer.Key).EndGroup(formSizeComplexUndoRedoCommand);
			ComponentHelper.Get(this._oldContainer.Key).MultipleSelection(this._list.Keys);
		}

		// Token: 0x06000ACD RID: 2765 RVA: 0x00035344 File Offset: 0x00033544
		public override void Clear()
		{
			if (this._list != null)
			{
				this._list.Clear();
			}
		}

		// Token: 0x04000403 RID: 1027
		private XmlElement _oldContainer;

		// Token: 0x04000404 RID: 1028
		private XmlElement _newContainer;

		// Token: 0x04000405 RID: 1029
		private int _posX = int.MinValue;

		// Token: 0x04000406 RID: 1030
		private int _posY = int.MinValue;

		// Token: 0x04000407 RID: 1031
		private Dictionary<XmlElement, DragComponentsUndoRedoCommand.ComponentDimension> _list = new Dictionary<XmlElement, DragComponentsUndoRedoCommand.ComponentDimension>();

		// Token: 0x02000131 RID: 305
		private class ComponentDimension
		{
			// Token: 0x170002D5 RID: 725
			// (get) Token: 0x06000ACE RID: 2766 RVA: 0x00035359 File Offset: 0x00033559
			// (set) Token: 0x06000ACF RID: 2767 RVA: 0x00035361 File Offset: 0x00033561
			public int OldX { get; private set; }

			// Token: 0x170002D6 RID: 726
			// (get) Token: 0x06000AD0 RID: 2768 RVA: 0x0003536A File Offset: 0x0003356A
			// (set) Token: 0x06000AD1 RID: 2769 RVA: 0x00035372 File Offset: 0x00033572
			public int OldY { get; private set; }

			// Token: 0x170002D7 RID: 727
			// (get) Token: 0x06000AD2 RID: 2770 RVA: 0x0003537B File Offset: 0x0003357B
			// (set) Token: 0x06000AD3 RID: 2771 RVA: 0x00035383 File Offset: 0x00033583
			public int OffsetX { get; private set; }

			// Token: 0x170002D8 RID: 728
			// (get) Token: 0x06000AD4 RID: 2772 RVA: 0x0003538C File Offset: 0x0003358C
			// (set) Token: 0x06000AD5 RID: 2773 RVA: 0x00035394 File Offset: 0x00033594
			public int OffsetY { get; private set; }

			// Token: 0x170002D9 RID: 729
			// (get) Token: 0x06000AD6 RID: 2774 RVA: 0x0003539D File Offset: 0x0003359D
			// (set) Token: 0x06000AD7 RID: 2775 RVA: 0x000353A5 File Offset: 0x000335A5
			public int OldIndex { get; set; }

			// Token: 0x170002DA RID: 730
			// (get) Token: 0x06000AD8 RID: 2776 RVA: 0x000353AE File Offset: 0x000335AE
			// (set) Token: 0x06000AD9 RID: 2777 RVA: 0x000353B6 File Offset: 0x000335B6
			public int NewIndex { get; set; }

			// Token: 0x06000ADA RID: 2778 RVA: 0x000353BF File Offset: 0x000335BF
			public ComponentDimension(int x, int y, int index, int offsetX, int offsetY)
			{
				this.OldX = x;
				this.OldY = y;
				this.OldIndex = index;
				this.NewIndex = -1;
				this.OffsetX = offsetX;
				this.OffsetY = offsetY;
			}
		}
	}
}
