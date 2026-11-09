using System;
using System.Collections.Generic;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.UndoRedoCommands;
using SpecDesignerCommon.ViewModel;
using UndoRedoFramework.Commands;

namespace SpecDesignerCommon.UndoRedo
{
	// Token: 0x02000122 RID: 290
	public class MoveComponentsUndoRedoCommand : AbstractComplexTriggerUndoRedoCommand
	{
		// Token: 0x06000A36 RID: 2614 RVA: 0x00032D14 File Offset: 0x00030F14
		public MoveComponentsUndoRedoCommand(List<XmlElement> list, MoveDirection direction, int offset)
		{
			this._direction = direction;
			this._offset = offset;
			list.Sort(delegate(XmlElement x, XmlElement y)
			{
				if (x == null)
				{
					if (y == null)
					{
						return 0;
					}
					return -1;
				}
				else
				{
					if (y == null)
					{
						return 0;
					}
					switch (direction)
					{
					case MoveDirection.Up:
					case MoveDirection.Down:
						return x.GridY - y.GridY;
					case MoveDirection.Left:
					case MoveDirection.Right:
						return x.GridX - y.GridX;
					default:
						return 0;
					}
				}
			});
			this._list = new Dictionary<XmlElement, MoveComponentsUndoRedoCommand.ComponentDimension>();
			foreach (XmlElement xmlElement in list)
			{
				if (null == this._key)
				{
					this._key = xmlElement.Key;
				}
				MoveComponentsUndoRedoCommand.ComponentDimension componentDimension = new MoveComponentsUndoRedoCommand.ComponentDimension(xmlElement.GridX, xmlElement.GridY);
				this._list.Add(xmlElement, componentDimension);
			}
		}

		// Token: 0x06000A37 RID: 2615 RVA: 0x00032DE8 File Offset: 0x00030FE8
		public override void Undo()
		{
			if (this._list == null)
			{
				return;
			}
			FormSizeComplexUndoRedoCommand formSizeComplexUndoRedoCommand = new FormSizeComplexUndoRedoCommand(this);
			SettingManager.Get().GetUndoRedoManager(this._key).StartGroup(formSizeComplexUndoRedoCommand);
			foreach (KeyValuePair<XmlElement, MoveComponentsUndoRedoCommand.ComponentDimension> keyValuePair in this._list)
			{
				keyValuePair.Key.GridX = keyValuePair.Value.X;
				keyValuePair.Key.GridY = keyValuePair.Value.Y;
			}
			SettingManager.Get().GetUndoRedoManager(this._key).EndGroup(formSizeComplexUndoRedoCommand);
			ComponentHelper.Get(this._key).MultipleSelection(this._list.Keys);
		}

		// Token: 0x06000A38 RID: 2616 RVA: 0x00032EBC File Offset: 0x000310BC
		public override void Execute()
		{
			if (this._list == null)
			{
				return;
			}
			FormSizeComplexUndoRedoCommand formSizeComplexUndoRedoCommand = new FormSizeComplexUndoRedoCommand(this);
			SettingManager.Get().GetUndoRedoManager(this._key).StartGroup(formSizeComplexUndoRedoCommand);
			foreach (KeyValuePair<XmlElement, MoveComponentsUndoRedoCommand.ComponentDimension> keyValuePair in this._list)
			{
				XmlElement key = keyValuePair.Key;
				switch (this._direction)
				{
				case MoveDirection.Up:
					key.GridY -= this._offset;
					break;
				case MoveDirection.Down:
					key.GridY += this._offset;
					break;
				case MoveDirection.Left:
					key.GridX -= this._offset;
					break;
				case MoveDirection.Right:
					key.GridX += this._offset;
					break;
				}
			}
			SettingManager.Get().GetUndoRedoManager(this._key).EndGroup(formSizeComplexUndoRedoCommand);
			ComponentHelper.Get(this._key).MultipleSelection(this._list.Keys);
		}

		// Token: 0x06000A39 RID: 2617 RVA: 0x00032FE0 File Offset: 0x000311E0
		public override void Clear()
		{
			if (this._list != null)
			{
				this._list.Clear();
			}
		}

		// Token: 0x040003DB RID: 987
		private MoveDirection _direction;

		// Token: 0x040003DC RID: 988
		private int _offset = -1;

		// Token: 0x040003DD RID: 989
		private Dictionary<XmlElement, MoveComponentsUndoRedoCommand.ComponentDimension> _list;

		// Token: 0x040003DE RID: 990
		private PackageKey _key;

		// Token: 0x02000123 RID: 291
		private class ComponentDimension
		{
			// Token: 0x170002B0 RID: 688
			// (get) Token: 0x06000A3A RID: 2618 RVA: 0x00032FF5 File Offset: 0x000311F5
			// (set) Token: 0x06000A3B RID: 2619 RVA: 0x00032FFD File Offset: 0x000311FD
			public int X { get; private set; }

			// Token: 0x170002B1 RID: 689
			// (get) Token: 0x06000A3C RID: 2620 RVA: 0x00033006 File Offset: 0x00031206
			// (set) Token: 0x06000A3D RID: 2621 RVA: 0x0003300E File Offset: 0x0003120E
			public int Y { get; private set; }

			// Token: 0x06000A3E RID: 2622 RVA: 0x00033017 File Offset: 0x00031217
			public ComponentDimension(int x, int y)
			{
				this.X = x;
				this.Y = y;
			}
		}
	}
}
